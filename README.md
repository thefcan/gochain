# gochain

A blockchain written from scratch in **Go** — proof of work, persistence, a UTXO
transaction model, **ECDSA wallets** and a peer-to-peer network. Built
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
- [ ] Phase 6 — Transaction signing & verification
- [ ] Phase 7 — P2P network (stretch)

## Architecture
```
gochain/
├── cmd/gochain/      # CLI: createwallet, createblockchain, getbalance, send, ...
└── internal/
    ├── wallet/       # ECDSA keys + Base58Check addresses + wallet file
    ├── tx/           # UTXO transactions: inputs, outputs, coinbase
    ├── block/        # Block (carries transactions) + serialization
    ├── pow/          # proof of work: mining + validation
    └── chain/        # persistent UTXO blockchain (BoltDB)
```

## Wallets & addresses
Wallets are **ECDSA** (P-256) key pairs. An address is the Bitcoin-style
**Base58Check** encoding of `version + RIPEMD160(SHA256(pubkey)) + checksum`, so
it carries a self-verifying checksum (`ValidateAddress` rejects typos). Wallets
are stored in a local file.

## Transactions (UTXO model)
Coins live in **unspent transaction outputs (UTXOs)**, like Bitcoin. A transfer
spends existing outputs as inputs and creates new outputs — the payment plus
*change* back to the sender. Balance is the sum of an address's unspent outputs.

## CLI / demo
```bash
go build -o gochain ./cmd/gochain

A=$(./gochain createwallet | awk '{print $NF}')   # real ECDSA address
B=$(./gochain createwallet | awk '{print $NF}')
./gochain createblockchain -address "$A"          # genesis reward (10) to A
./gochain send -from "$A" -to "$B" -amount 4
./gochain getbalance -address "$A"                # 6 (change)
./gochain getbalance -address "$B"                # 4
# GOCHAIN_DB / GOCHAIN_WALLET configure the storage paths
```

## Tests & benchmark
```bash
go test -race ./...                            # unit, crypto, UTXO and persistence tests
go test -bench=Run -benchmem ./internal/pow    # mining benchmark
```
