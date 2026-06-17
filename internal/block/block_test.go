package block

import "testing"

func TestNewSetsFields(t *testing.T) {
	b := New("hello", []byte("prevhash"))

	if string(b.Data) != "hello" {
		t.Errorf("Data = %q, want %q", b.Data, "hello")
	}
	if string(b.PrevBlockHash) != "prevhash" {
		t.Errorf("PrevBlockHash = %q, want %q", b.PrevBlockHash, "prevhash")
	}
	if len(b.Hash) != 32 {
		t.Errorf("Hash length = %d, want 32 (SHA-256)", len(b.Hash))
	}
	if b.Timestamp == 0 {
		t.Error("Timestamp was not set")
	}
}

func TestDifferentDataProducesDifferentHash(t *testing.T) {
	a := New("alpha", []byte{})
	b := New("beta", []byte{})
	if string(a.Hash) == string(b.Hash) {
		t.Error("different data produced the same hash")
	}
}

func TestGenesisHasNoPrevHash(t *testing.T) {
	g := NewGenesis()
	if len(g.PrevBlockHash) != 0 {
		t.Errorf("genesis PrevBlockHash = %x, want empty", g.PrevBlockHash)
	}
	if len(g.Hash) != 32 {
		t.Errorf("genesis Hash length = %d, want 32", len(g.Hash))
	}
}
