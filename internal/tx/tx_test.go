package tx

import "testing"

func TestNewCoinbaseTX(t *testing.T) {
	cb, err := NewCoinbaseTX("alice", "")
	if err != nil {
		t.Fatalf("NewCoinbaseTX: %v", err)
	}
	if !cb.IsCoinbase() {
		t.Error("IsCoinbase() = false, want true")
	}
	if len(cb.Vout) != 1 || cb.Vout[0].Value != subsidy {
		t.Errorf("coinbase output = %+v, want value %d", cb.Vout, subsidy)
	}
	if !cb.Vout[0].CanBeUnlockedWith("alice") {
		t.Error("coinbase output not unlockable by its recipient")
	}
	if len(cb.ID) == 0 {
		t.Error("ID was not set")
	}
}

func TestRegularTxIsNotCoinbase(t *testing.T) {
	regular := &Transaction{
		Vin:  []TXInput{{Txid: []byte("abc"), Vout: 0, ScriptSig: "alice"}},
		Vout: []TXOutput{{Value: 5, ScriptPubKey: "bob"}},
	}
	if regular.IsCoinbase() {
		t.Error("IsCoinbase() = true for a regular transaction")
	}
}

func TestUnlocking(t *testing.T) {
	in := TXInput{ScriptSig: "alice"}
	if !in.CanUnlockOutputWith("alice") || in.CanUnlockOutputWith("bob") {
		t.Error("TXInput unlocking is incorrect")
	}
	out := TXOutput{ScriptPubKey: "bob"}
	if !out.CanBeUnlockedWith("bob") || out.CanBeUnlockedWith("alice") {
		t.Error("TXOutput unlocking is incorrect")
	}
}
