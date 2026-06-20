# gochain

A blockchain written from scratch in **Go** — proof of work, persistence, a
signed UTXO transaction model, ECDSA wallets and **peer-to-peer networking**.
Built incrementally, with tests, CI and Docker, to the engineering standard of a
production service.

## Why Go?
Go is the dominant language of blockchain **infrastructure** — go-ethereum (geth),
the Cosmos SDK / Tendermint and Hyperledger Fabric are all written in Go — so this
implementation maps directly to the work blockchain teams actually do.

## Status — all phases complete
- [x] **Phase 1 — Blocks & in-memory chain** (SHA-256 linking)
- [x] **Phase 2 — Proof of Work** (Hashcash mining, difficulty, validation)
- [x] **Phase 3 — Persistence (BoltDB) + CLI**
- [x] **Phase 4 — UTXO transactions** (coinbase, transfers with change, balances)
- [x] **Phase 5 — Wallets & addresses** (ECDSA + Base58Check)
- [x] **Phase 6 — Signed transactions** (ECDSA sign/verify, tamper-rejecting)
- [x] **Phase 7 — Peer-to-peer network** (TCP block sync between nodes)

## Architecture
```
gochain/
├── cmd/gochain/      # CLI: wallets, chain operations, node + sync
└── internal/
    ├── wallet/       # ECDSA keys, Base58Check addresses, wallet file
    ├── tx/           # signed UTXO transactions (sign + verify)
    ├── block/        # Block (transactions) + serialization
    ├── pow/          # proof of work: mining + validation
    ├── chain/        # persistent UTXO blockchain (BoltDB)
    └── network/      # TCP peer-to-peer block replication
```
Clean, acyclic dependencies: `wallet`/`tx` → `block` → `pow` → `chain` → `network`.

## What it does
- **Proof of Work**: every block is mined to a difficulty target; tampering breaks it.
- **UTXO model**: coins live in unspent outputs; transfers create payment + change.
- **Cryptographic security**: outputs are locked to a public-key hash; spending
  requires an **ECDSA signature** verified before mining (forgeries rejected).
- **Persistence**: the chain is stored in an embedded **BoltDB** file.
- **P2P sync**: a fresh node downloads and verifies a peer's chain over **TCP**.

## CLI
```bash
go build -o gochain ./cmd/gochain

A=$(./gochain createwallet | awk '{print $NF}')
B=$(./gochain createwallet | awk '{print $NF}')
./gochain createblockchain -address "$A"          # genesis reward (10) to A
./gochain send -from "$A" -to "$B" -amount 4      # A's wallet signs; chain verifies
./gochain getbalance -address "$A"                # 6 (change)

# Peer-to-peer: serve on one node, sync from another
./gochain startnode -port 3000                    # node A serves its chain
GOCHAIN_DB=node-b.db ./gochain sync -peer localhost:3000   # node B replicates it
```
Storage paths are configurable via `GOCHAIN_DB` and `GOCHAIN_WALLET`.

## Tests & benchmark
```bash
go test -race ./...                            # unit, crypto, signing, persistence, P2P
go test -bench=Run -benchmem ./internal/pow    # mining benchmark
```
The suite covers proof of work, the UTXO model, ECDSA sign/verify (including
forgery and tamper rejection), persistence across restarts, and TCP block sync.
