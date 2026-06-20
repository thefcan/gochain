package wallet

import "errors"

// PubKeyHashFromAddress extracts the 20-byte public-key hash embedded in a
// Base58Check address (dropping the version byte and the trailing checksum).
func PubKeyHashFromAddress(address string) ([]byte, error) {
	full := Base58Decode([]byte(address))
	if len(full) < addrChecksumLen+1 {
		return nil, errors.New("invalid address")
	}
	return full[1 : len(full)-addrChecksumLen], nil
}
