// Command gochain is a minimal blockchain CLI. Phase 1 builds an in-memory chain
// and prints it; later phases add proof of work, persistence, transactions and
// wallets.
package main

import (
	"fmt"

	"github.com/thefcan/gochain/internal/chain"
)

func main() {
	bc := chain.New()
	bc.AddBlock("Send 1 coin to Furkan")
	bc.AddBlock("Send 2 more coins to Furkan")

	for _, b := range bc.Blocks() {
		fmt.Printf("Prev. hash: %x\n", b.PrevBlockHash)
		fmt.Printf("Data:       %s\n", b.Data)
		fmt.Printf("Hash:       %x\n\n", b.Hash)
	}
}
