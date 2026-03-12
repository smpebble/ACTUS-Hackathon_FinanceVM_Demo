// Package solidity provides Solidity code generation for ACTUS contracts
package solidity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/actus-go/pkg/codegen"
)

// Generator implements code generation for Solidity (Ethereum/EVM)
type Generator struct {
	renderer codegen.TemplateRenderer
}

// NewGenerator creates a new Solidity generator
func NewGenerator() *Generator {
	return &Generator{
		renderer: codegen.NewTemplateRenderer(),
	}
}

// Generate produces Solidity smart contract code from ContractIR
func (g *Generator) Generate(ctx context.Context, ir *codegen.ContractIR, opts codegen.GeneratorOptions) (*codegen.GeneratedCode, error) {
	// Validate first
	if err := g.Validate(ir); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Set defaults
	if opts.Version == "" {
		opts.Version = "0.8.20"
	}
	if opts.ContractName == "" {
		opts.ContractName = ir.Metadata.ContractName
	}

	// Load templates
	templateDir := opts.CustomTemplateDir
	if templateDir == "" {
		templateDir = "templates/solidity"
	}

	// Generate main contract
	mainContract, err := g.generateMainContract(ir, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate main contract: %w", err)
	}

	result := &codegen.GeneratedCode{
		Language:        codegen.LanguageSolidity,
		MainFile:        mainContract,
		AdditionalFiles: make([]codegen.CodeFile, 0),
		Metadata: codegen.GenerationMetadata{
			GeneratedAt:        time.Now().Format(time.RFC3339),
			GeneratorVersion:   "1.0.0",
			SourceContractType: ir.Metadata.ACTUSType,
			Warnings:           make([]string, 0),
			Dependencies:       g.getDependencies(ir),
		},
	}

	// Generate interface if needed
	if opts.IncludeDocumentation {
		interfaceFile := g.generateInterface(ir, opts)
		result.AdditionalFiles = append(result.AdditionalFiles, interfaceFile)
	}

	// Generate tests if requested
	if opts.IncludeTests {
		testFile := g.generateTests(ir, opts)
		result.AdditionalFiles = append(result.AdditionalFiles, testFile)
	}

	return result, nil
}

// generateMainContract generates the main smart contract file
func (g *Generator) generateMainContract(ir *codegen.ContractIR, opts codegen.GeneratorOptions) (codegen.CodeFile, error) {
	var sb strings.Builder

	// SPDX License
	sb.WriteString(fmt.Sprintf("// SPDX-License-Identifier: %s\n", ir.Metadata.License))
	sb.WriteString(fmt.Sprintf("pragma solidity ^%s;\n\n", opts.Version))

	// Contract description
	if ir.Metadata.Description != "" {
		sb.WriteString(fmt.Sprintf("/**\n * @title %s\n * @dev %s\n", opts.ContractName, ir.Metadata.Description))
		sb.WriteString(fmt.Sprintf(" * @notice ACTUS %s contract implementation\n", ir.Metadata.ACTUSType))
		sb.WriteString(" */\n")
	}

	// Contract declaration
	sb.WriteString(fmt.Sprintf("contract %s {\n", opts.ContractName))

	// State variables
	sb.WriteString("\n    // State Variables\n")
	for _, sv := range ir.StateVariables {
		g.writeStateVariable(&sb, sv)
	}

	// Events
	if len(ir.Events) > 0 {
		sb.WriteString("\n    // Events\n")
		for _, evt := range ir.Events {
			g.writeEvent(&sb, evt)
		}
	}

	// Constructor
	sb.WriteString("\n    // Constructor\n")
	g.writeConstructor(&sb, ir, opts)

	// Functions
	sb.WriteString("\n    // Functions\n")
	for _, fn := range ir.Functions {
		g.writeFunction(&sb, fn)
	}

	// Close contract
	sb.WriteString("}\n")

	return codegen.CodeFile{
		Filename: fmt.Sprintf("%s.sol", opts.ContractName),
		Content:  sb.String(),
		FileType: codegen.FileTypeContract,
	}, nil
}

// writeStateVariable writes a state variable declaration
func (g *Generator) writeStateVariable(sb *strings.Builder, sv codegen.StateVariable) {
	modifier := ""
	if sv.Immutable {
		modifier = " immutable"
	}

	visibility := string(sv.Visibility)
	if visibility == "" {
		visibility = "public"
	}

	sb.WriteString(fmt.Sprintf("    %s %s%s %s;\n",
		g.mapDataType(sv.Type),
		visibility,
		modifier,
		sv.Name,
	))
}

// writeEvent writes an event declaration
func (g *Generator) writeEvent(sb *strings.Builder, evt codegen.Event) {
	params := make([]string, len(evt.Parameters))
	for i, p := range evt.Parameters {
		indexed := ""
		if p.Indexed {
			indexed = " indexed"
		}
		params[i] = fmt.Sprintf("%s%s %s", g.mapDataType(p.Type), indexed, p.Name)
	}

	sb.WriteString(fmt.Sprintf("    event %s(%s);\n", evt.Name, strings.Join(params, ", ")))
}

// writeConstructor writes the constructor function
func (g *Generator) writeConstructor(sb *strings.Builder, ir *codegen.ContractIR, opts codegen.GeneratorOptions) {
	sb.WriteString("    constructor(")

	// Constructor parameters from state variables
	params := make([]string, 0)
	for _, sv := range ir.StateVariables {
		if !sv.Immutable && sv.InitialValue == nil {
			dt := g.mapDataType(sv.Type)
			location := g.dataLocation(dt)
			params = append(params, fmt.Sprintf("%s%s _%s", dt, location, sv.Name))
		}
	}
	sb.WriteString(strings.Join(params, ", "))
	sb.WriteString(") {\n")

	// Initialize state variables
	for _, sv := range ir.StateVariables {
		if sv.InitialValue != nil {
			sb.WriteString(fmt.Sprintf("        %s = %v;\n", sv.Name, sv.InitialValue))
		} else if !sv.Immutable {
			sb.WriteString(fmt.Sprintf("        %s = _%s;\n", sv.Name, sv.Name))
		}
	}

	sb.WriteString("    }\n")
}

// writeFunction writes a function declaration
func (g *Generator) writeFunction(sb *strings.Builder, fn codegen.Function) {
	// Skip constructor as it's handled separately
	if fn.Type == codegen.FunctionTypeConstructor {
		return
	}

	// Function signature
	params := make([]string, len(fn.Parameters))
	for i, p := range fn.Parameters {
		dt := g.mapDataType(p.Type)
		location := g.dataLocation(dt)
		params[i] = fmt.Sprintf("%s%s %s", dt, location, p.Name)
	}

	visibility := string(fn.Visibility)
	if visibility == "" {
		visibility = "public"
	}

	fnType := ""
	if fn.Type == codegen.FunctionTypeView {
		fnType = " view"
	} else if fn.Type == codegen.FunctionTypePure {
		fnType = " pure"
	} else if fn.Type == codegen.FunctionTypePayable {
		fnType = " payable"
	}

	sb.WriteString(fmt.Sprintf("\n    function %s(%s) %s%s",
		fn.Name,
		strings.Join(params, ", "),
		visibility,
		fnType,
	))

	// Return values
	if len(fn.Returns) > 0 {
		returns := make([]string, len(fn.Returns))
		for i, r := range fn.Returns {
			returns[i] = g.mapDataType(r.Type)
		}
		sb.WriteString(fmt.Sprintf(" returns (%s)", strings.Join(returns, ", ")))
	}

	sb.WriteString(" {\n")

	// Require checks
	for _, req := range fn.Body.RequireChecks {
		sb.WriteString(fmt.Sprintf("        require(%s, \"%s\");\n", req.Condition, req.Message))
	}

	// Function body operations
	for _, op := range fn.Body.Operations {
		g.writeOperation(sb, op)
	}

	sb.WriteString("    }\n")
}

// writeOperation writes a single operation
func (g *Generator) writeOperation(sb *strings.Builder, op codegen.Operation) {
	switch op.Type {
	case codegen.OpTypeReturn:
		sb.WriteString(fmt.Sprintf("        return %s;\n", op.Target))
	case codegen.OpTypeAssign:
		sb.WriteString(fmt.Sprintf("        %s = %v;\n", op.Target, op.Args[0]))
	case codegen.OpTypeEmit:
		args := make([]string, len(op.Args))
		for i, arg := range op.Args {
			args[i] = fmt.Sprint(arg)
		}
		sb.WriteString(fmt.Sprintf("        emit %s(%s);\n", op.Target, strings.Join(args, ", ")))
	default:
		sb.WriteString(fmt.Sprintf("        // TODO: implement %s operation\n", op.Type))
	}
}

// mapDataType maps IR data type to Solidity type
func (g *Generator) mapDataType(dt codegen.DataType) string {
	if dt.CustomType != "" {
		return dt.CustomType
	}

	switch dt.BaseType {
	case "uint256", "uint", "int", "int256", "address", "bool", "string", "bytes":
		return dt.BaseType
	case "decimal":
		return "uint256" // Map decimal to uint256 with implicit scaling
	default:
		return "uint256"
	}
}

// dataLocation returns the Solidity data location modifier for reference types.
// Value types (uint256, address, bool, etc.) need no location.
// Reference types (string, bytes, arrays, structs) require "memory" in function/constructor params.
func (g *Generator) dataLocation(solType string) string {
	switch solType {
	case "string", "bytes":
		return " memory"
	default:
		// Check for array types (e.g., uint256[], bytes32[])
		if strings.HasSuffix(solType, "[]") {
			return " memory"
		}
		return ""
	}
}

// generateInterface generates an interface file
func (g *Generator) generateInterface(ir *codegen.ContractIR, opts codegen.GeneratorOptions) codegen.CodeFile {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("// SPDX-License-Identifier: %s\n", ir.Metadata.License))
	sb.WriteString(fmt.Sprintf("pragma solidity ^%s;\n\n", opts.Version))
	sb.WriteString(fmt.Sprintf("interface I%s {\n", opts.ContractName))

	// Events
	for _, evt := range ir.Events {
		g.writeEvent(&sb, evt)
	}

	// Function signatures only
	for _, fn := range ir.Functions {
		if fn.Visibility == codegen.VisibilityPublic || fn.Visibility == codegen.VisibilityExternal {
			params := make([]string, len(fn.Parameters))
			for i, p := range fn.Parameters {
				params[i] = fmt.Sprintf("%s %s", g.mapDataType(p.Type), p.Name)
			}

			sb.WriteString(fmt.Sprintf("    function %s(%s) external", fn.Name, strings.Join(params, ", ")))

			if len(fn.Returns) > 0 {
				returns := make([]string, len(fn.Returns))
				for i, r := range fn.Returns {
					returns[i] = g.mapDataType(r.Type)
				}
				sb.WriteString(fmt.Sprintf(" returns (%s)", strings.Join(returns, ", ")))
			}
			sb.WriteString(";\n")
		}
	}

	sb.WriteString("}\n")

	return codegen.CodeFile{
		Filename: fmt.Sprintf("I%s.sol", opts.ContractName),
		Content:  sb.String(),
		FileType: codegen.FileTypeInterface,
	}
}

// generateTests generates a test file
func (g *Generator) generateTests(ir *codegen.ContractIR, opts codegen.GeneratorOptions) codegen.CodeFile {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("// SPDX-License-Identifier: %s\n", ir.Metadata.License))
	sb.WriteString(fmt.Sprintf("pragma solidity ^%s;\n\n", opts.Version))
	sb.WriteString("import \"forge-std/Test.sol\";\n")
	sb.WriteString(fmt.Sprintf("import \"./%s.sol\";\n\n", opts.ContractName))

	sb.WriteString(fmt.Sprintf("contract %sTest is Test {\n", opts.ContractName))
	sb.WriteString(fmt.Sprintf("    %s public contractInstance;\n\n", opts.ContractName))

	sb.WriteString("    function setUp() public {\n")
	sb.WriteString("        // TODO: Initialize contract instance\n")
	sb.WriteString("    }\n\n")

	sb.WriteString("    function testInitialization() public {\n")
	sb.WriteString("        // TODO: Test contract initialization\n")
	sb.WriteString("    }\n")

	sb.WriteString("}\n")

	return codegen.CodeFile{
		Filename: fmt.Sprintf("%s.t.sol", opts.ContractName),
		Content:  sb.String(),
		FileType: codegen.FileTypeTest,
	}
}

// Validate checks if the IR can be generated for Solidity
func (g *Generator) Validate(ir *codegen.ContractIR) error {
	if ir == nil {
		return fmt.Errorf("IR cannot be nil")
	}

	if ir.Metadata.ContractName == "" {
		return fmt.Errorf("contract name is required")
	}

	return nil
}

// GetSupportedFeatures returns ACTUS features supported by Solidity generator
func (g *Generator) GetSupportedFeatures() []string {
	return []string{
		"PAM",   // Principal at Maturity
		"LAM",   // Linear Amortizer
		"ANN",   // Annuity
		"SWAPS", // Interest Rate Swaps
		"OPTNS", // Options
	}
}

// GetLanguage returns the target language
func (g *Generator) GetLanguage() codegen.TargetLanguage {
	return codegen.LanguageSolidity
}

// getDependencies returns required dependencies for the generated code
func (g *Generator) getDependencies(ir *codegen.ContractIR) []codegen.Dependency {
	deps := []codegen.Dependency{}

	// Add common dependencies
	deps = append(deps, codegen.Dependency{
		Name:    "OpenZeppelin Contracts",
		Version: "^5.0.0",
		Source:  "https://github.com/OpenZeppelin/openzeppelin-contracts",
	})

	return deps
}
