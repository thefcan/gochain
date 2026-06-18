# gochain

A blockchain written from scratch in **Go** — proof of work, persistence, a UTXO
transaction model, ECDSA wallets and a peer-to-peer network. Built incrementally,
with tests, CI and Docker, to the same engineering standard as a production
service.

## Why Go?
Go is the dominant language of blockchain **infrastructure** — go-ethereum (geth),
the Cosmos SDK / Tendermint and Hyperledger Fabric are all written in Go — so this
implementation maps directly to the work blockchain teams actually do.

## Status (built in phases)
- [x] **Phase 1 — Blocks & in-memory chain** (SHA-256 linking)
- [x] **Phase 2 — Proof of Work** (Hashcash mining, difficulty, validation)
- [x] **Phase 3 — Persistence (BoltDB) + CLI**
- [ ] Phase 4 — UTXO transactions
- [ ] Phase 5 — Wallets & addresses (ECDSA, Base58)
- [ ] Phase 6 — Transaction signing & verification
- [ ] Phase 7 — P2P network (stretch)

## Architecture
```
gochain/
├── cmd/gochain/      # CLI: addblock, printchain
└── internal/
    ├── block/        # Block type + gob serialization
    ├── pow/          # proof of work: mining + validation
    └── chain/        # persistent chain (BoltDB) + iterator
```
Layers are deliberately separated — `block` is data, `pow` is the algorithm,
`chain` orchestrates and persists — so there are no circular dependencies.

## Persistence
Blocks are stored in an embedded **BoltDB** file (keyed by hash, with a pointer
to the tip), so the chain survives restarts. Iteration walks from the tip back to
genesis.

## CLI
```bash
go build -o gochain ./cmd/gochain

./gochain addblock -data "Send 1 coin to Furkan"   # mine & append a block
./gochain printchain                               # print the chain (tip -> genesis)
# database path is configurable: GOCHAIN_DB=/path/to/chain.db ./gochain printchain
```

## Tests & benchmark
```bash
go test -race ./...                            # unit + persistence tests
go test -bench=Run -benchmem ./internal/pow    # mining benchmark
```
