package network

import (
	"bytes"
	"encoding/gob"
	"testing"
)

// FuzzDecodeMessage feeds arbitrary bytes to the peer-message decoder — the same
// gob path used by handle() when reading a request off a connection. The decoder
// must never panic on hostile input; a decode error is an acceptable outcome.
//
// Run: go test -run '^$' -fuzz FuzzDecodeMessage -fuzztime 30s ./internal/network/
func FuzzDecodeMessage(f *testing.F) {
	var buf bytes.Buffer
	_ = gob.NewEncoder(&buf).Encode(message{Command: cmdInv, Hashes: [][]byte{{0x01, 0x02, 0x03}}})
	f.Add(buf.Bytes())
	f.Add([]byte{})
	f.Add([]byte{0xff, 0x81, 0x03, 0x01})

	f.Fuzz(func(t *testing.T, data []byte) {
		var m message
		_ = gob.NewDecoder(bytes.NewReader(data)).Decode(&m)
		// Property: decoding attacker-controlled bytes must not panic or crash.
		// A returned error is fine; we only guard the process against faults.
	})
}
