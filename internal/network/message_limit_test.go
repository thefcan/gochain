package network

import (
	"bytes"
	"encoding/gob"
	"testing"
)

// TestDecodeMessageRejectsOversized asserts the peer-message decoder refuses to
// read past maxMessageBytes, so a hostile peer cannot drive it to allocate
// unbounded memory. It shrinks the cap for the test and restores it afterwards.
func TestDecodeMessageRejectsOversized(t *testing.T) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(message{Command: cmdInv, Hashes: [][]byte{{1, 2, 3}}}); err != nil {
		t.Fatalf("encode: %v", err)
	}

	// Under the real cap the message decodes cleanly.
	if _, err := decodeMessage(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatalf("decode under cap: %v", err)
	}

	// With a cap smaller than the message, decoding must fail rather than read
	// (and allocate for) the whole, potentially unbounded, stream.
	saved := maxMessageBytes
	maxMessageBytes = 4
	defer func() { maxMessageBytes = saved }()
	if _, err := decodeMessage(bytes.NewReader(buf.Bytes())); err == nil {
		t.Fatal("decode over cap: got nil error, want failure")
	}
}
