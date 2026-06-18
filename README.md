# gochain

A blockchain written from scratch in **Go** — proof of work, a UTXO transaction
model, ECDSA wallets and a peer-to-peer network. Built incrementally, with tests,
CI and Docker, to the same engineering standard as a production service.

## Why Go?
Go is the dominant language of blockchain **infrastructure** — go-ethereum (geth),
the Cosmos SDK / Tendermint and Hyperledger Fabric are all written in Go — so this
implementation maps directly to the work blockchain teams actually do.

## Status (built in phases)
- [x] **Phase 1 — Blocks & in-memory chain** (SHA-256 linking)
- [x] **Phase 2 — Proof of Work** (Hashcash mining, difficulty, validation)
- [ ] Phase 3 — Persistence (BoltDB) + CLI
- [ ] Phase 4 — UTXO transactions
- [ ] Phase 5 — Wallets & addresses (ECDSA, Base58)
- [ ] Phase 6 — Transaction signing & verification
- [ ] Phase 7 — P2P network (stretch)

## Architecture
```
gochain/
├── cmd/gochain/      # CLI entry point
└── internal/
    ├── block/        # Block type (data)
    ├── pow/          # proof of work: mining + validation
    └── chain/        # blockchain orchestration
```
Layers are deliberately separated — `block` is plain data, `pow` is the
algorithm, `chain` orchestrates — so there are no circular dependencies.

## Proof of Work
Each block is *mined*: the miner searches for a nonce whose SHA-256 block hash,
read as a big integer, falls below a difficulty target (`TargetBits` leading zero
bits). Tampering with any field invalidates the recorded proof.

```text
Data:       Genesis Block
Hash:       00000ca30f0c536b4618c1288d37e422b38121250e21edc033abd47c94adb4c3
Nonce:      4619
PoW valid:  true
```

## Run
```bash
go run ./cmd/gochain
```

## Tests & benchmark
```bash
go test -race ./...                       # unit tests with the race detector
go test -bench=Run -benchmem ./internal/pow   # mining benchmark
```
