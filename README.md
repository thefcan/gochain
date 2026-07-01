# gochain

[![CI](https://github.com/thefcan/gochain/actions/workflows/ci.yml/badge.svg)](https://github.com/thefcan/gochain/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![tests](https://img.shields.io/badge/tests-race%2C%20property%2C%20fuzz-4c1)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)

*A blockchain written from scratch in **Go** — proof of work, persistence, a
signed UTXO transaction model, ECDSA wallets and TCP peer-to-peer sync.*

Built to the engineering standard of a production service: property-based and
fuzz tests, a security-reviewed network layer, static analysis, Docker and CI.

## Why Go?
Go is the dominant language of blockchain **infrastructure** — go-ethereum (geth),
the Cosmos SDK / Tendermint and Hyperledger Fabric are all written in Go — so this
implementation maps directly to the work blockchain teams actually do.

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
Wallet keys use the modern `crypto/ecdsa` encoding APIs (Go 1.25+).

## What it does
- **Proof of Work**: every block is mined to a difficulty target; tampering breaks it.
- **UTXO model**: coins live in unspent outputs; transfers create payment + change.
- **Cryptographic security**: outputs are locked to a public-key hash; spending
  requires an **ECDSA signature** verified before mining (forgeries rejected).
- **Persistence**: the chain is stored in an embedded **BoltDB** file.
- **P2P sync**: a fresh node downloads a peer's chain over **TCP** and validates
  every block — proof of work **and** transaction signatures — before storing it.

## CLI
```bash
go build -o gochain ./cmd/gochain

A=$(./gochain createwallet | awk '{print $NF}')
B=$(./gochain createwallet | awk '{print $NF}')
./gochain createblockchain -address "$A"          # genesis reward (10) to A
./gochain send -from "$A" -to "$B" -amount 4      # A's wallet signs; chain verifies
./gochain getbalance -address "$A"                # 6 (change)

# Peer-to-peer: serve on one node, sync from another
./gochain startnode -port 3000
GOCHAIN_DB=node-b.db ./gochain sync -peer localhost:3000
```

## Docker
A multi-stage build produces a tiny (~12 MB) distroless, non-root image:
```bash
docker build -t gochain .
docker run --rm -e GOCHAIN_WALLET=/tmp/w.dat gochain createwallet

# or run a node with docker-compose (chain stored in a named volume)
docker compose up --build
```

## Testing & hardening

Correctness and robustness are verified at several levels. The unit, property and
integration suites run under the Go **race detector** in CI on every push; the
fuzz targets run on demand (see below).

- **Unit & integration tests** across every package — crypto, signing,
  persistence, and end-to-end TCP peer sync.
- **Property-based tests** ([`rapid`](https://github.com/flyingmutant/rapid)),
  asserting invariants over thousands of generated inputs:
  - transaction hashing is deterministic and independent of the `ID` field;
  - **sign → verify soundness** — a valid signature verifies, and tampering with
    the signature or any signed output is rejected;
  - block serialization is round-trip idempotent;
  - **value conservation & no double-spend** — a transfer preserves total value,
    and an overspend is rejected with balances left intact.
- **Native Go fuzzing** of the two decoders that parse untrusted bytes — the block
  deserializer and the P2P wire-message decoder — proving they never panic on
  hostile input (their seed corpora also run as part of the normal `-race` suite).

### Security review

A focused review of the network- and consensus-facing code found and closed the
issues below, each now guarded by a regression test:

| Area | Issue | Fix |
|------|-------|-----|
| P2P decode | Unbounded `gob` decode of peer input allowed a memory-exhaustion (OOM) DoS | 16 MiB read cap via `io.LimitReader` |
| P2P I/O | No socket deadlines — a silent peer could pin a goroutine indefinitely | Per-exchange read/write deadlines + dial timeout |
| Transactions | An input's output index was used without a lower-bound check → panic on a crafted negative index | Full bounds check; the transaction is rejected cleanly |
| Consensus | Received blocks were trusted on valid proof of work **alone** — cheap at this difficulty, so forged transactions could be injected | Every transaction in a received block is signature-verified against the chain before storage |

### Running the checks
```bash
go test -race ./...                                          # unit + property + integration
golangci-lint run ./...                                      # static analysis (.golangci.yml)
go test -run '^$' -fuzz FuzzDeserializeBlock -fuzztime 30s ./internal/block/
go test -run '^$' -fuzz FuzzDecodeMessage    -fuzztime 30s ./internal/network/
go test -bench=Run -benchmem ./internal/pow                 # mining benchmark
```

## License

[MIT](LICENSE) © 2026 Furkan Karafil
