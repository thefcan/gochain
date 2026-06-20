// Command gochain is a blockchain CLI: ECDSA wallets, a signed UTXO transaction
// model and proof of work. Paths are configurable via GOCHAIN_DB (default
// gochain.db) and GOCHAIN_WALLET (default wallet.dat).
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/thefcan/gochain/internal/chain"
	"github.com/thefcan/gochain/internal/pow"
	"github.com/thefcan/gochain/internal/wallet"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "createwallet":
		err = cmdCreateWallet(os.Args[2:])
	case "listaddresses":
		err = cmdListAddresses(os.Args[2:])
	case "createblockchain":
		err = cmdCreate(os.Args[2:])
	case "getbalance":
		err = cmdBalance(os.Args[2:])
	case "send":
		err = cmdSend(os.Args[2:])
	case "printchain":
		err = cmdPrint(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gochain:", err)
		os.Exit(1)
	}
}

func dbPath() string {
	if p := os.Getenv("GOCHAIN_DB"); p != "" {
		return p
	}
	return "gochain.db"
}

func walletFile() string {
	if p := os.Getenv("GOCHAIN_WALLET"); p != "" {
		return p
	}
	return "wallet.dat"
}

func cmdCreateWallet(args []string) error {
	ws, err := wallet.LoadWallets(walletFile())
	if err != nil {
		return err
	}
	addr, err := ws.CreateWallet()
	if err != nil {
		return err
	}
	if err := ws.Save(walletFile()); err != nil {
		return err
	}
	fmt.Printf("new address: %s\n", addr)
	return nil
}

func cmdListAddresses(args []string) error {
	ws, err := wallet.LoadWallets(walletFile())
	if err != nil {
		return err
	}
	addrs := ws.GetAddresses()
	if len(addrs) == 0 {
		fmt.Println("no wallets yet; run: gochain createwallet")
		return nil
	}
	for _, a := range addrs {
		fmt.Println(a)
	}
	return nil
}

func cmdCreate(args []string) error {
	fs := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	address := fs.String("address", "", "address to receive the genesis reward")
	_ = fs.Parse(args)
	if *address == "" {
		return errors.New("createblockchain: -address is required")
	}
	if !wallet.ValidateAddress(*address) {
		return fmt.Errorf("invalid address: %s", *address)
	}
	bc, err := chain.CreateBlockchain(dbPath(), *address)
	if err != nil {
		return err
	}
	defer bc.Close()
	fmt.Printf("blockchain created; genesis reward to %s\n", *address)
	return nil
}

func cmdBalance(args []string) error {
	fs := flag.NewFlagSet("getbalance", flag.ExitOnError)
	address := fs.String("address", "", "address to query")
	_ = fs.Parse(args)
	if *address == "" {
		return errors.New("getbalance: -address is required")
	}
	if !wallet.ValidateAddress(*address) {
		return fmt.Errorf("invalid address: %s", *address)
	}
	bc, err := chain.Open(dbPath())
	if err != nil {
		return err
	}
	defer bc.Close()
	bal, err := bc.Balance(*address)
	if err != nil {
		return err
	}
	fmt.Printf("balance of %s: %d\n", *address, bal)
	return nil
}

func cmdSend(args []string) error {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	from := fs.String("from", "", "sender address")
	to := fs.String("to", "", "recipient address")
	amount := fs.Int("amount", 0, "amount to send")
	_ = fs.Parse(args)
	if *from == "" || *to == "" || *amount <= 0 {
		return errors.New("send: -from, -to and a positive -amount are required")
	}
	if !wallet.ValidateAddress(*from) || !wallet.ValidateAddress(*to) {
		return errors.New("send: invalid -from or -to address")
	}

	ws, err := wallet.LoadWallets(walletFile())
	if err != nil {
		return err
	}
	w, ok := ws.GetWallet(*from)
	if !ok {
		return fmt.Errorf("no wallet for sender %s (run: gochain createwallet)", *from)
	}

	bc, err := chain.Open(dbPath())
	if err != nil {
		return err
	}
	defer bc.Close()
	if err := bc.Send(w, *to, *amount); err != nil {
		return err
	}
	fmt.Printf("sent %d from %s to %s\n", *amount, *from, *to)
	return nil
}

func cmdPrint(args []string) error {
	bc, err := chain.Open(dbPath())
	if err != nil {
		return err
	}
	defer bc.Close()

	it := bc.Iterator()
	for {
		b, err := it.Next()
		if err != nil {
			return err
		}
		if b == nil {
			break
		}
		fmt.Printf("Block %x  (PoW valid: %t)\n", b.Hash, pow.New(b).Validate())
		for _, t := range b.Transactions {
			fmt.Printf("  TX %x  coinbase=%t\n", t.ID, t.IsCoinbase())
			for _, out := range t.Vout {
				fmt.Printf("    out: %d -> pkh:%x\n", out.Value, out.PubKeyHash)
			}
		}
		fmt.Println()
	}
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, `  gochain createwallet                     generate a new wallet/address`)
	fmt.Fprintln(os.Stderr, `  gochain listaddresses                    list wallet addresses`)
	fmt.Fprintln(os.Stderr, `  gochain createblockchain -address X      create a chain (genesis reward to X)`)
	fmt.Fprintln(os.Stderr, `  gochain getbalance -address X            print X's balance`)
	fmt.Fprintln(os.Stderr, `  gochain send -from A -to B -amount N      transfer N from A to B (A's wallet signs)`)
	fmt.Fprintln(os.Stderr, `  gochain printchain                       print the chain`)
}
