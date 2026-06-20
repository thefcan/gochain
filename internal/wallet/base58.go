package wallet

import (
	"bytes"
	"math/big"
)

var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

// Base58Encode encodes input using the Bitcoin Base58 alphabet.
func Base58Encode(input []byte) []byte {
	var result []byte
	x := new(big.Int).SetBytes(input)
	base := big.NewInt(int64(len(b58Alphabet)))
	zero := big.NewInt(0)
	mod := new(big.Int)

	for x.Cmp(zero) != 0 {
		x.DivMod(x, base, mod)
		result = append(result, b58Alphabet[mod.Int64()])
	}
	reverse(result)

	// Leading zero bytes are encoded as the first alphabet character.
	for _, b := range input {
		if b != 0x00 {
			break
		}
		result = append([]byte{b58Alphabet[0]}, result...)
	}
	return result
}

// Base58Decode reverses Base58Encode. It returns nil on an invalid character.
func Base58Decode(input []byte) []byte {
	result := big.NewInt(0)
	base := big.NewInt(int64(len(b58Alphabet)))
	for _, c := range input {
		idx := bytes.IndexByte(b58Alphabet, c)
		if idx == -1 {
			return nil
		}
		result.Mul(result, base)
		result.Add(result, big.NewInt(int64(idx)))
	}
	decoded := result.Bytes()

	var zeros int
	for _, c := range input {
		if c != b58Alphabet[0] {
			break
		}
		zeros++
	}
	return append(bytes.Repeat([]byte{0x00}, zeros), decoded...)
}

func reverse(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}
