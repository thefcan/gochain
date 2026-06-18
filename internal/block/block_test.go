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

func TestSerializeRoundTrip(t *testing.T) {
	orig := New("payload", []byte("prev"))
	orig.Hash = []byte("somehash")
	orig.Nonce = 42

	data, err := orig.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	got, err := Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	if string(got.Data) != "payload" || got.Nonce != 42 ||
		string(got.Hash) != "somehash" || string(got.PrevBlockHash) != "prev" {
		t.Errorf("round-trip mismatch: got %+v", got)
	}
}
