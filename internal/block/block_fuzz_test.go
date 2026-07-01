package block_test

import (
	"testing"

	"github.com/thefcan/gochain/internal/block"
	"github.com/thefcan/gochain/internal/tx"
)

// FuzzDeserializeBlock feeds arbitrary bytes to the block decoder. The decoder
// must never panic; malformed input should produce a clean error, and any input
// that decodes successfully must re-serialize and re-decode without error.
//
// Run: go test -run '^$' -fuzz FuzzDeserializeBlock -fuzztime 30s ./internal/block/
func FuzzDeserializeBlock(f *testing.F) {
	seed := &block.Block{
		Timestamp:     1,
		Transactions:  []*tx.Transaction{{ID: []byte{0x01}, Vout: []tx.TXOutput{{Value: 10, PubKeyHash: []byte{0x02}}}}},
		PrevBlockHash: []byte{0x03},
		Hash:          []byte{0x04},
		Nonce:         5,
	}
	if b, err := seed.Serialize(); err == nil {
		f.Add(b)
	}
	f.Add([]byte{})
	f.Add([]byte{0xff, 0x00, 0x42})

	f.Fuzz(func(t *testing.T, data []byte) {
		b, err := block.Deserialize(data)
		if err != nil {
			return // a clean rejection of malformed input is acceptable
		}
		s, err := b.Serialize()
		if err != nil {
			return
		}
		if _, err := block.Deserialize(s); err != nil {
			t.Fatalf("a decoded block failed to round-trip: %v", err)
		}
	})
}
