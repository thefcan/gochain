package block

import "testing"

func TestNewReturnsUnminedBlock(t *testing.T) {
	b := New("hello", []byte("prevhash"))

	if string(b.Data) != "hello" {
		t.Errorf("Data = %q, want %q", b.Data, "hello")
	}
	if string(b.PrevBlockHash) != "prevhash" {
		t.Errorf("PrevBlockHash = %q, want %q", b.PrevBlockHash, "prevhash")
	}
	if b.Timestamp == 0 {
		t.Error("Timestamp was not set")
	}
	// Hash and Nonce are set later by proof of work.
	if b.Hash != nil {
		t.Errorf("Hash = %x, want nil before mining", b.Hash)
	}
	if b.Nonce != 0 {
		t.Errorf("Nonce = %d, want 0 before mining", b.Nonce)
	}
}

func TestGenesisData(t *testing.T) {
	g := NewGenesis()
	if string(g.Data) != "Genesis Block" {
		t.Errorf("genesis Data = %q, want %q", g.Data, "Genesis Block")
	}
	if len(g.PrevBlockHash) != 0 {
		t.Errorf("genesis PrevBlockHash = %x, want empty", g.PrevBlockHash)
	}
}
