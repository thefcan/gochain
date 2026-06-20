# gochain

A blockchain written from scratch in **Go** — proof of work, persistence, a UTXO
transaction model, ECDSA wallets and **digitally signed transactions**. Built
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
- [x] **Phase 5 — Wallets & addresses** (ECDSA + Base58Check)
- [x] **Phase 6 — Signed transactions** (ECDSA sign/verify, tamper-rejecting)
- [ ] Phase 7 — P2P network (stretch)

## Architecture
```
gochain/
├── cmd/gochain/      # CLI: createwallet, createblockchain, getbalance, send, ...
└── internal/
    ├── wallet/       # ECDSA keys, Base58Check addresses, wallet file
    ├── tx/           # signed UTXO transactions (sign + verify)
    ├── block/        # Block (carries transactions) + serialization
    ├── pow/          # proof of work: mining + validation
    └── chain/        # persistent UTXO blockchain (BoltDB)
```

## Security model
Each output is locked to a **public-key hash**. Spending it requires an input
carrying the spender's public key and an **ECDSA signature** over the transaction.
Before mining a block, the chain **verifies every transaction**: a forged or
tampered signature is rejected (see the `tx` and `chain` tests). Signatures are
fixed-width (`r||s`, 64 bytes) so verification is deterministic.

## CLI / demo
```bash
go build -o gochain ./cmd/gochain

A=$(./gochain createwallet | awk '{print $NF}')
B=$(./gochain createwallet | awk '{print $NF}')
./gochain createblockchain -address "$A"      # genesis reward (10) to A
./gochain send -from "$A" -to "$B" -amount 4  # A's wallet signs; chain verifies
./gochain getbalance -address "$A"            # 6 (change)
./gochain getbalance -address "$B"            # 4
```

## Tests & benchmark
```bash
go test -race ./...                            # unit, crypto, signing and persistence tests
go test -bench=Run -benchmem ./internal/pow    # mining benchmark
```
