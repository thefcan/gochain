# gochain

A blockchain written from scratch in **Go** — proof of work, persistence, a
**UTXO transaction model**, ECDSA wallets and a peer-to-peer network. Built
incrementally, with tests, CI and Docker, to the engineering standard of a
production service.

## Why Go?
Go is the dominant language of blockchain **infrastructure** — go-ethereum (geth),
the Cosmos SDK / Tendermint and Hyperledger Fabric are all written in Go — so this
implementation maps directly to the work blockchain teams actually do.

## Status (built in phases)
- [x] **Phase 1 — Blocks & in-memory chain** (SHA-256 linking)
- [x] **Phase 2 — Proof of Work** (Hashcash mining, difficulty, validation)
- [x] **Phase 3 — Persistence (BoltDB) + CLI**
- [x] **Phase 4 — UTXO transactions** (coinbase, transfers with change, balances)
- [ ] Phase 5 — Wallets & addresses (ECDSA, Base58)
- [ ] Phase 6 — Transaction signing & verification
- [ ] Phase 7 — P2P network (stretch)

## Architecture
```
gochain/
├── cmd/gochain/      # CLI: createblockchain, getbalance, send, printchain
└── internal/
    ├── tx/           # UTXO transactions: inputs, outputs, coinbase
    ├── block/        # Block (carries transactions) + serialization
    ├── pow/          # proof of work: mining + validation
    └── chain/        # persistent UTXO blockchain (BoltDB)
```
Layers are deliberately separated with no circular dependencies:
`tx` ← `block` ← `pow`; `chain` orchestrates them.

## Transactions (UTXO model)
Coins live in **unspent transaction outputs (UTXOs)**, exactly like Bitcoin. A
transfer spends existing outputs as inputs and creates new outputs — the payment
plus *change* back to the sender. An address's balance is the sum of its unspent
outputs. (Walking the chain tip→genesis lets a spend be seen before the output it
spends, so spent outputs are excluded correctly — no double counting.)

## CLI / demo
```bash
go build -o gochain ./cmd/gochain

./gochain createblockchain -address Furkan   # genesis reward (10) to Furkan
./gochain send -from Furkan -to Ali -amount 4
./gochain getbalance -address Furkan         # 6  (4 sent, 6 change)
./gochain getbalance -address Ali            # 4
./gochain printchain                         # inspect blocks and transactions
# database path is configurable with GOCHAIN_DB
```

## Tests & benchmark
```bash
go test -race ./...                            # unit, UTXO and persistence tests
go test -bench=Run -benchmem ./internal/pow    # mining benchmark
```
