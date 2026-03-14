# CLAUDE.md — SeaShell Codebase Guide

SeaShell is a minimal, educational cryptocurrency implementation in Go. It includes a blockchain engine, a UTXO-based transaction model, ECDSA wallets, and a CLI. This document helps AI assistants understand the codebase structure and conventions.

---

## Project Overview

| Attribute | Value |
|-----------|-------|
| Language | Go 1.18 |
| Module | `github.com/goozt/seashell` |
| Build | `make build` |
| Binary output | `bin/seashell` |
| Consensus | Proof of Work (difficulty 12) |
| Tx model | UTXO (unspent transaction output) |
| Crypto | ECDSA P256 + SHA256 + SHA3-256 |
| Storage | BadgerDB (blockchain) + GOB file (wallets) |

---

## Directory Structure

```
seashell/
├── main.go              # Entry point — calls cli.SeashellCli()
├── go.mod               # Module: github.com/goozt/seashell, Go 1.18
├── go.sum               # Dependency checksums
├── Makefile             # Build targets: install, build
├── init-env.sh          # Downloads Go 1.18 locally if needed
├── LICENSE
├── README.md
├── blockchain/          # Core blockchain logic
│   ├── block.go         # Block struct + GOB serialization
│   ├── chain.go         # BlockChain: DB ops, UTXO queries, validation
│   ├── proof.go         # Proof of Work (SHA256, difficulty 12)
│   ├── transaction.go   # Transaction: create, sign, verify (ECDSA)
│   ├── txio.go          # TxInput / TxOutput structs
│   └── utils.go         # Handle() error helper, hex utils
├── wallet/              # Wallet and address management
│   ├── wallet.go        # Wallet struct with ECDSA key pair
│   ├── db.go            # WalletDB: GOB persistence of wallet map
│   ├── address.go       # Address generation + Base58 + checksum
│   └── utils.go         # Handle() error helper
└── cli/                 # Command-line interface
    ├── cli.go           # Arg parsing, usage, command dispatch
    ├── chainCmd.go      # create, balance, send, list commands
    └── walletCmd.go     # wallet, walletlist commands
```

---

## Build and Run

```bash
# First-time setup (installs Go 1.18 locally if not on system)
make build

# Run the CLI
bin/seashell <command>
```

### Available CLI Commands

| Command | Flags | Description |
|---------|-------|-------------|
| `create` | `-a ADDRESS` | Initialize blockchain with genesis block rewarding ADDRESS |
| `balance` | `-a ADDRESS` | Show coin balance for ADDRESS |
| `send` | `-from F -to T -amount N` | Send N coins from F to T |
| `list` | — | Print all blocks in the chain |
| `wallet` | — | Generate and save a new wallet/address |
| `walletlist` | — | List all saved wallet addresses |

---

## Key Architectural Patterns

### 1. UTXO Model
Coin ownership is tracked by unspent transaction outputs, not account balances.

- `chain.FindUnspentTransactions(pubKeyHash)` — scans the chain for UTXOs
- `chain.FindSpendableOutputs(pubKeyHash, amount)` — collects enough UTXOs to fund a send
- `chain.FindUTXO(pubKeyHash)` — returns all unspent outputs for an address

A `TxOutput` is spent when it is referenced by a `TxInput` elsewhere in the chain.

### 2. Proof of Work
- File: `blockchain/proof.go`
- Constant: `Difficulty = 12` (top 12 bits of SHA256 hash must be zero)
- `NewProof(block).Run()` — mines nonce until hash condition is met
- `pow.Validate()` — re-runs hash to verify a block

### 3. ECDSA Transaction Signing
- Key pair: `elliptic.P256()` curve
- Signing: each `TxInput` is signed with the sender's private key
- Verification: `transaction.Verify(prevTxs)` checks all input signatures against the referenced output's `PubKeyHash`
- Signature is stored as `r || s` (concatenated big-endian bytes)

### 4. Address Format
```
PublicKey
  → SHA256
  → SHA3-256       (PubKeyHash)
  → version(0x00) + PubKeyHash + checksum(4 bytes)
  → Base58Encode
  → Address string
```
- `wallet.ValidateAddress(addr)` — decodes and verifies checksum

### 5. Persistence
**Blockchain** — BadgerDB at `./db/blocks`
| Key | Value |
|-----|-------|
| `[]byte(blockHash)` | GOB-encoded `Block` |
| `[]byte("lh")` | last block hash |

**Wallets** — GOB file at `./db/wallets.data`
- `WalletDB.Wallets` is a `map[string]*Wallet` keyed by address string

Both storage formats use Go's `encoding/gob`. When adding new fields to `Block`, `Transaction`, or `Wallet`, ensure backward compatibility or handle deserialization errors.

---

## Constants (Hardcoded)

| Constant | Location | Value | Purpose |
|----------|----------|-------|---------|
| `Difficulty` | `blockchain/proof.go` | `12` | PoW difficulty |
| `ChecksumLength` | `wallet/address.go` | `4` | Bytes for address checksum |
| `version` | `wallet/address.go` | `0x00` | Address version byte |
| `dbPath` | `blockchain/chain.go` | `"./db/blocks"` | BlockChain DB directory |
| `dbFile` | `blockchain/chain.go` | `"./db/blocks/MANIFEST"` | Existence check file |
| `walletFile` | `wallet/db.go` | `"./db/wallets.data"` | Wallet persistence file |
| `genesisData` | `blockchain/chain.go` | `"Initial transaction from Genesis"` | Genesis coinbase data |

---

## Error Handling Convention

Both `blockchain/utils.go` and `wallet/utils.go` define:

```go
func Handle(err error) {
    if err != nil {
        log.Panic(err)
    }
}
```

All errors are handled by calling `Handle(err)`, which panics on non-nil errors. There is no structured error propagation — panics propagate up and are not recovered. When adding new code, follow this same pattern for consistency.

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/dgraph-io/badger/v3` | Embedded key-value store for blockchain |
| `github.com/mr-tron/base58` | Base58 encoding for wallet addresses |
| `golang.org/x/crypto` | SHA3-256 for public key hashing |

---

## Development Notes

- **No tests exist.** There are no `*_test.go` files. When adding features, consider adding tests in a `_test.go` file alongside the relevant package.
- **No linter config.** Run `go vet ./...` manually for static analysis.
- **No CI/CD.** All builds are local via `make build`.
- **Database is local.** The `./db/` directory is gitignored. Run `create` before any other chain command.
- **No environment variables.** All paths and constants are hardcoded. Do not introduce `os.Getenv` without also providing a `.env.example`.
- **Single-process only.** BadgerDB does not support concurrent access from multiple processes.

---

## Workflow for Common Tasks

### Adding a new CLI command
1. Define the flag set and logic in `cli/chainCmd.go` or `cli/walletCmd.go`
2. Register the command in `cli/cli.go` inside `CommandLine.Run()`
3. Update the usage string at the top of `cli/cli.go`

### Adding a new transaction type
1. Add the type/fields to `blockchain/transaction.go`
2. Update `Sign()` and `Verify()` if the signing domain changes
3. Update `chain.go` callers (`CreateTransaction`, `MineBlock`, etc.)

### Changing block structure
1. Edit `blockchain/block.go`
2. GOB encoding is field-order sensitive — existing `./db/blocks` data will be incompatible; delete and reinitialize with `create`

### Changing address format
1. Edit `wallet/address.go`
2. Existing wallet addresses stored in `./db/wallets.data` will be invalid — regenerate wallets

---

## Git

- Main development branch: `claude/add-claude-documentation-NP937`
- Remote: configured as local proxy
- Commits: use clear, imperative messages (e.g., `Add transaction fee support`)
- Gitignored: `/db`, `/tmp`, `/bin`
