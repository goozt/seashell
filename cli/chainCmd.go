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

	walletDB, err := wallet.CreateWalletDB()
	blockchain.HandleFatalErrors(err)
	w := walletDB.GetWallet(address)

	chain := blockchain.InitBlockChain(false, address, w.PublicKey, w.PrivateKey)
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

	walletDB, err := wallet.CreateWalletDB()
	blockchain.HandleFatalErrors(err)
	w := walletDB.GetWallet(from)

	chain := blockchain.ContinueBlockChain(false, "")
	defer chain.Close()

	tx := blockchain.NewTransaction(from, to, amount, chain)
	chain.AddBlock([]*blockchain.Transaction{tx}, w.PublicKey, w.PrivateKey)

	fmt.Println("Added new block")
}

func (cli *CommandLine) list() {
	chain := blockchain.ContinueBlockChain(false, "")
	defer chain.Close()
	iter := chain.Iterator()
	for {
		block := iter.Next()

		fmt.Printf("Block %x\n", block.Hash)
		fmt.Printf("  Height: %d\n", block.Height)
		fmt.Printf("  Timestamp: %d\n", block.Timestamp)
		fmt.Printf("  PreviousHash: %x\n", block.PrevHash)
		fmt.Printf("  Lead Validator: %x\n", block.LeadValidator())
		fmt.Printf("  Signatures: %d\n", len(block.Signatures))
		fmt.Printf("  Valid PoA: %s\n", strconv.FormatBool(blockchain.ValidateBlock(block, chain.Database)))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Println()
		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) addvalidator(address string) {
	if !wallet.ValidateAddress(address) {
		log.Fatalln("address is not valid")
	}

	walletDB, err := wallet.CreateWalletDB()
	blockchain.HandleFatalErrors(err)
	w := walletDB.GetWallet(address)

	chain := blockchain.ContinueBlockChain(false, "")
	defer chain.Close()

	chain.AddValidator(w.PublicKey)
	fmt.Printf("Validator added: %s\n", address)
}
