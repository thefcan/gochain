package chain

import (
	"testing"

	"github.com/thefcan/gochain/internal/pow"
)

func TestNewHasGenesis(t *testing.T) {
	bc := New()
	if got := len(bc.Blocks()); got != 1 {
		t.Fatalf("new chain length = %d, want 1 (genesis only)", got)
	}
}

func TestAddBlockLinksToTip(t *testing.T) {
	bc := New()
	bc.AddBlock("first")
	bc.AddBlock("second")

	blocks := bc.Blocks()
	if len(blocks) != 3 {
		t.Fatalf("chain length = %d, want 3", len(blocks))
	}
	for i := 1; i < len(blocks); i++ {
		if string(blocks[i].PrevBlockHash) != string(blocks[i-1].Hash) {
			t.Errorf("block %d does not link to block %d's hash", i, i-1)
		}
	}
}

func TestEveryBlockIsValidlyMined(t *testing.T) {
	bc := New()
	bc.AddBlock("a")
	bc.AddBlock("b")
	for _, b := range bc.Blocks() {
		if !pow.New(b).Validate() {
			t.Errorf("block %x failed proof-of-work validation", b.Hash)
		}
	}
}
