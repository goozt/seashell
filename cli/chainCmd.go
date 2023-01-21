package cli

import (
	"fmt"
	"log"
	"strconv"

	"github.com/goozt/seashell/blockchain"
	"github.com/goozt/seashell/wallet"
)

func (cli *CommandLine) create(address string) {
	if !wallet.ValidateAddress(address) {
		log.Fatalln("address is not valid")
	}

	chain := blockchain.InitBlockChain(false, address)
	defer chain.Close()
	fmt.Println("New blockchain created")
}

func (cli *CommandLine) balance(address string) {
	if !wallet.ValidateAddress(address) {
		log.Fatalln("address is not valid")
	}

	chain := blockchain.ContinueBlockChain(false, address)
	defer chain.Close()

	balance := 0
	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-wallet.ChecksumLength]
	UTXOs := chain.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of %s: %d\n", address, balance)
}

func (cli *CommandLine) send(from, to string, amount int) {
	if !wallet.ValidateAddress(from) {
		log.Fatalln("from address is not valid")
	}
	if !wallet.ValidateAddress(to) {
		log.Fatalln("to address is not valid")
	}
	chain := blockchain.ContinueBlockChain(false, "")
	defer chain.Close()

	tx := blockchain.NewTransaction(from, to, amount, chain)
	chain.AddBlock([]*blockchain.Transaction{tx})

	fmt.Println("Added new block")
}

func (cli *CommandLine) list() {
	chain := blockchain.ContinueBlockChain(false, "")
	defer chain.Close()
	iter := chain.Iterator()
	for {
		block := iter.Next()

		fmt.Printf("Block %x\n", block.Hash)
		fmt.Printf("  Timestamp: %d\n", block.Timestamp)
		fmt.Printf("  PreviousHash: %x\n", block.PrevHash)

		pow := blockchain.NewProof(block)
		fmt.Printf("  Valid PoW: %s\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Println()
		if len(block.PrevHash) == 0 {
			break
		}
	}
}
