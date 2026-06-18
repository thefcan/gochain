// Command gochain is a blockchain CLI. Phase 3 persists the chain to a BoltDB
// file and exposes `addblock` and `printchain` subcommands. The database path
// can be overridden with GOCHAIN_DB (default: gochain.db).
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/thefcan/gochain/internal/chain"
	"github.com/thefcan/gochain/internal/pow"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	dbPath := os.Getenv("GOCHAIN_DB")
	if dbPath == "" {
		dbPath = "gochain.db"
	}
	bc, err := chain.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open chain: %v\n", err)
		os.Exit(1)
	}
	defer bc.Close()

	switch os.Args[1] {
	case "addblock":
		fs := flag.NewFlagSet("addblock", flag.ExitOnError)
		data := fs.String("data", "", "block data")
		_ = fs.Parse(os.Args[2:])
		if *data == "" {
			fmt.Fprintln(os.Stderr, "addblock: -data is required")
			os.Exit(1)
		}
		if err := bc.AddBlock(*data); err != nil {
			fmt.Fprintf(os.Stderr, "add block: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("block added")
	case "printchain":
		if err := printChain(bc); err != nil {
			fmt.Fprintf(os.Stderr, "print chain: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func printChain(bc *chain.Blockchain) error {
	it := bc.Iterator()
	for {
		b, err := it.Next()
		if err != nil {
			return err
		}
		if b == nil {
			break
		}
		fmt.Printf("Hash:       %x\n", b.Hash)
		fmt.Printf("Prev. hash: %x\n", b.PrevBlockHash)
		fmt.Printf("Data:       %s\n", b.Data)
		fmt.Printf("Nonce:      %d\n", b.Nonce)
		fmt.Printf("PoW valid:  %t\n\n", pow.New(b).Validate())
	}
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, `  gochain addblock -data "..."   mine and append a block`)
	fmt.Fprintln(os.Stderr, "  gochain printchain             print the chain (tip -> genesis)")
}
