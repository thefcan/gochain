// Command gochain is a minimal blockchain CLI. Phase 2 mines each block with
// proof of work and prints the chain with its nonces and validity.
package main

import (
	"fmt"

	"github.com/thefcan/gochain/internal/chain"
	"github.com/thefcan/gochain/internal/pow"
)

func main() {
	bc := chain.New()
	bc.AddBlock("Send 1 coin to Furkan")
	bc.AddBlock("Send 2 more coins to Furkan")

	for _, b := range bc.Blocks() {
		fmt.Printf("Prev. hash: %x\n", b.PrevBlockHash)
		fmt.Printf("Data:       %s\n", b.Data)
		fmt.Printf("Hash:       %x\n", b.Hash)
		fmt.Printf("Nonce:      %d\n", b.Nonce)
		fmt.Printf("PoW valid:  %t\n\n", pow.New(b).Validate())
	}
}
