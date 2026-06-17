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
- [ ] Phase 2 — Proof of Work (mining, difficulty)
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
    ├── block/        # Block type + hashing
    └── chain/        # in-memory blockchain
```

## Run
```bash
go run ./cmd/gochain
```

## Tests
```bash
go test -race ./...
```
