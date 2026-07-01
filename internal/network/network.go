// Package network provides a minimal peer-to-peer protocol for replicating the
// blockchain between nodes over TCP. Each request is a short-lived connection
// carrying one gob-encoded message and its response.
package network

import (
	"encoding/gob"
	"errors"
	"io"
	"log"
	"net"
	"time"

	"github.com/thefcan/gochain/internal/block"
	"github.com/thefcan/gochain/internal/chain"
)

// Protocol commands.
const (
	cmdGetBlocks = "getblocks" // request all block hashes
	cmdInv       = "inv"       // response: block hashes
	cmdGetData   = "getdata"   // request a block by hash
	cmdBlock     = "block"     // response: a serialized block
)

type message struct {
	Command string
	Hashes  [][]byte
	Hash    []byte
	Block   []byte
}

// maxMessageBytes caps how many bytes we read for a single peer message, so a
// hostile peer cannot make the gob decoder allocate unbounded memory by
// declaring a huge length. It is a var (not a const) only so tests can shrink
// it; treat it as constant at runtime.
var maxMessageBytes int64 = 16 << 20 // 16 MiB

// ioTimeout bounds a single peer exchange, so a slow or silent peer cannot pin
// a goroutine and its connection open forever.
const ioTimeout = 30 * time.Second

// decodeMessage reads exactly one gob-encoded message from r, refusing to read
// past maxMessageBytes.
func decodeMessage(r io.Reader) (message, error) {
	var m message
	err := gob.NewDecoder(io.LimitReader(r, maxMessageBytes)).Decode(&m)
	return m, err
}

// Node serves a blockchain to peers and can sync from them.
type Node struct {
	bc *chain.Blockchain
}

// NewNode wraps a blockchain in a network node.
func NewNode(bc *chain.Blockchain) *Node { return &Node{bc: bc} }

// Listen starts serving peer requests on addr in the background and returns the
// listener so the caller can stop it with Close.
func (n *Node) Listen(addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	go n.serve(ln)
	return ln, nil
}

func (n *Node) serve(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed
		}
		go n.handle(conn)
	}
}

func (n *Node) handle(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(ioTimeout))

	msg, err := decodeMessage(conn)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			log.Printf("network: decode request: %v", err)
		}
		return
	}
	switch msg.Command {
	case cmdGetBlocks:
		hashes, err := n.bc.GetBlockHashes()
		if err != nil {
			log.Printf("network: getblocks: %v", err)
			return
		}
		if err := gob.NewEncoder(conn).Encode(message{Command: cmdInv, Hashes: hashes}); err != nil {
			log.Printf("network: send inv: %v", err)
		}
	case cmdGetData:
		b, err := n.bc.GetBlock(msg.Hash)
		if err != nil {
			log.Printf("network: getdata %x: %v", msg.Hash, err)
			return
		}
		ser, err := b.Serialize()
		if err != nil {
			log.Printf("network: serialize block: %v", err)
			return
		}
		if err := gob.NewEncoder(conn).Encode(message{Command: cmdBlock, Block: ser}); err != nil {
			log.Printf("network: send block: %v", err)
		}
	default:
		log.Printf("network: unknown command %q", msg.Command)
	}
}

// SyncFrom downloads any blocks the local chain is missing from peerAddr and
// returns how many were added.
func (n *Node) SyncFrom(peerAddr string) (int, error) {
	inv, err := request(peerAddr, message{Command: cmdGetBlocks})
	if err != nil {
		return 0, err
	}

	added := 0
	for _, h := range inv.Hashes { // genesis -> tip order
		has, err := n.bc.HasBlock(h)
		if err != nil {
			return added, err
		}
		if has {
			continue
		}
		resp, err := request(peerAddr, message{Command: cmdGetData, Hash: h})
		if err != nil {
			return added, err
		}
		b, err := block.Deserialize(resp.Block)
		if err != nil {
			return added, err
		}
		if err := n.bc.AddReceivedBlock(b); err != nil {
			return added, err
		}
		added++
	}
	return added, nil
}

// request opens a connection to peerAddr, sends req and returns the response.
func request(peerAddr string, req message) (message, error) {
	conn, err := net.DialTimeout("tcp", peerAddr, ioTimeout)
	if err != nil {
		return message{}, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(ioTimeout))
	if err := gob.NewEncoder(conn).Encode(req); err != nil {
		return message{}, err
	}
	resp, err := decodeMessage(conn)
	if err != nil {
		return message{}, err
	}
	return resp, nil
}
