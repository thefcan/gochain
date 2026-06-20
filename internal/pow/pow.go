// Package pow implements Hashcash-style Proof of Work: finding a nonce such that
// a block's SHA-256 hash, read as a big integer, is below a difficulty target.
package pow

import (
	"bytes"
	"crypto/sha256"
	"math"
	"math/big"
	"strconv"

	"github.com/thefcan/gochain/internal/block"
)

// TargetBits is the mining difficulty: the number of leading zero bits required
// in a block's hash. Higher is exponentially harder.
const TargetBits = 16

const maxNonce = math.MaxInt64

// ProofOfWork couples a block with the target its hash must satisfy.
type ProofOfWork struct {
	block  *block.Block
	target *big.Int
}

// New returns a ProofOfWork for b at the configured difficulty.
func New(b *block.Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-TargetBits))
	return &ProofOfWork{block: b, target: target}
}

// prepareData serialises the block's headers (including a Merkle-style hash of
// its transactions) together with a candidate nonce.
func (pow *ProofOfWork) prepareData(nonce int) []byte {
	return bytes.Join([][]byte{
		pow.block.PrevBlockHash,
		pow.block.HashTransactions(),
		[]byte(strconv.FormatInt(pow.block.Timestamp, 10)),
		[]byte(strconv.Itoa(TargetBits)),
		[]byte(strconv.Itoa(nonce)),
	}, []byte{})
}

// Run mines the block: it increments the nonce until the hash is below the
// target, returning the winning nonce and the resulting hash.
func (pow *ProofOfWork) Run() (int, []byte) {
	var hashInt big.Int
	var hash [32]byte

	for nonce := 0; nonce < maxNonce; nonce++ {
		hash = sha256.Sum256(pow.prepareData(nonce))
		hashInt.SetBytes(hash[:])
		if hashInt.Cmp(pow.target) == -1 {
			return nonce, hash[:]
		}
	}
	return maxNonce, hash[:]
}

// Validate reports whether the block's recorded nonce satisfies the target.
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int
	hash := sha256.Sum256(pow.prepareData(pow.block.Nonce))
	hashInt.SetBytes(hash[:])
	return hashInt.Cmp(pow.target) == -1
}
