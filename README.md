# ACTUS-FVM Demo

**ACTUS Hackathon 2025 Finals**

Demonstrates the **FinanceVM (FVM)** platform integrating four open-source projects to tokenise financial assets on-chain with ACTUS-standard cashflow calculations.

---

## Integrated Stack

| Component | Role |
|---|---|
| **ACTUS Contract** | PAM / ANN / SWAPS contract engine, cashflow scheduling |
| **Financial Execution Kernel（FEK）** | Instrument lifecycle management (Draft → Issued → Active) |
| **ISO 20022** | Financial message generation (pacs.008, sese.023, camt.054) |
| **vLEI** | Verifiable Legal Entity Identity verification |
| **Shopspring/Decimal** | Zero-precision-loss monetary arithmetic |
| **Solidity Codegen** | Smart contract generation from ACTUS contract specs |

---

## Four Demo Scenarios

| # | Scenario | Contract | Highlight |
|---|---|---|---|
| 1 | **Stablecoin (TWDX)** | PAM | 1:1 TWD backing, zero precision loss, ERC-20 |
| 2 | **Corporate Bond (5Y)** | PAM | 3.5% coupon, Act/360, semi-annual, ERC-1400 |
| 3 | **SME Loan** | ANN | 4.5% fixed, 60 monthly amortisation payments |
| 4 | **Interest Rate Swap** | SWAPS | Fixed 3.0% vs floating TAIBOR+50bps, quarterly |

---

## Prerequisites

- **Go 1.21+** installed

---

## Quick Start — Web Demo (recommended)

```bash
git clone https://github.com/smpebble/ACTUS-Hackathon_FinanceVM_Demo
cd ACTUS-Hackathon_FinanceVM_Demo
go run cmd/server/main.go
```

Then open **http://localhost:8080** in your browser.

- Click a scenario card to run it individually
- Click **Run All Scenarios** to execute all four in sequence
- Results appear in tabbed sections: **Cashflows · Precision · ISO 20022 · vLEI · Solidity**

---

## Quick Start — CLI Demo

```bash
# From the ACTUS-FVM project root:
go run cmd/demo/main.go
# or
go run main.go
```

Prints all four scenarios to the console with formatted tables.

---

## Project Structure

```
ACTUS-FVM/
├── cmd/
│   ├── demo/main.go         # CLI demo runner
│   └── server/main.go       # Web server entry point
├── internal/
│   ├── api/server.go        # HTTP API (GET /api/health, /api/scenarios, POST /api/run)
│   ├── adapter/             # Wrappers for ACTUS-GO, FVM, vLEI, ISO20022, Codegen
│   ├── demo/                # Runner + console output formatting
│   ├── model/               # Unified domain types (Instrument, CashFlowEvent, …)
│   └── scenarios/           # 4 financial scenarios (stablecoin, bond, loan, derivative)
├── web/
│   └── index.html           # Single-page web demo (Tailwind CSS, Vanilla JS)
├── generated/               # Output: generated Solidity contracts
│   ├── stablecoin/
│   ├── bond/
│   ├── loan/
│   └── derivative/
└── Docs/                    # Full documentation (Traditional Chinese + English)
```

---

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/health` | Health check → `{"status":"ok"}` |
| `GET` | `/api/scenarios` | Scenario metadata list |
| `POST` | `/api/run` | Run scenario(s) → full results JSON |

**POST /api/run** body:

```json
{ "scenario": "all" }
{ "scenario": "stablecoin" }
{ "scenario": "bond" }
{ "scenario": "loan" }
{ "scenario": "derivative" }
```

---

## Run Tests

```bash
go test ./...
```

---

## Key Technical Decisions

1. **Direct Go imports** — no HTTP micro-services; single `go run` starts everything
2. **In-memory mock** — FVM engine uses in-memory storage (no PostgreSQL required)
3. **vLEI mock mode** — no GLEIF testnet connection required
4. **Built-in ISO 20022** — uses FVM's built-in message generator (no Claude API)
5. **shopspring/decimal** — all monetary calculations use exact decimal arithmetic

---

## GitHub

https://github.com/smpebble/ACTUS-Hackathon_FinanceVM_Demo
