const solc = require('solc');
const fs = require('fs');

async function compileSolidity() {
    let input = '';
    process.stdin.setEncoding('utf-8');

    for await (const chunk of process.stdin) {
        input += chunk;
    }

    if (!input || input.trim() === '') {
        console.error(JSON.stringify({ error: "Empty input code." }));
        process.exit(1);
    }

    try {
        const inputSpec = {
            language: 'Solidity',
            sources: {
                'Contract.sol': {
                    content: input
                }
            },
            settings: {
                outputSelection: {
                    '*': {
                        '*': ['*']
                    }
                }
            }
        };

        const output = JSON.parse(solc.compile(JSON.stringify(inputSpec)));

        let result = {
            success: true,
            errors: [],
            contracts: []
        };

        if (output.errors) {
            output.errors.forEach(err => {
                if (err.severity === 'error') {
                    result.success = false;
                }
                result.errors.push(err.formattedMessage);
            });
        }

        if (output.contracts && output.contracts['Contract.sol']) {
            for (let contractName in output.contracts['Contract.sol']) {
                let contractMeta = output.contracts['Contract.sol'][contractName];

                // Truncate somewhat to keep payload manageable for frontend
                let evmInfo = contractMeta.evm || {};
                let byteCode = (evmInfo.bytecode && evmInfo.bytecode.object) ? evmInfo.bytecode.object : "";

                if (byteCode.length > 200) {
                    byteCode = byteCode.substring(0, 200) + "...[TRUNCATED]";
                }

                result.contracts.push({
                    name: contractName,
                    abi: contractMeta.abi,
                    bytecode: byteCode
                });
            }
        }

        console.log(JSON.stringify(result, null, 2));

    } catch (e) {
        console.error(JSON.stringify({ error: e.toString() }));
        process.exit(1);
    }
}

compileSolidity();
