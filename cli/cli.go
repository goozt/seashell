package cli

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/goozt/seashell/blockchain"
)

type CommandLine struct{}

func (cli *CommandLine) usage() {
	fmt.Println("Usage:")
	fmt.Println(" balance -a ADDRESS")
	fmt.Println(" create -a ADDRESS")
	fmt.Println(" send -from ADDRESS -to ADDRESS -amount VALUE")
	fmt.Println(" list")
	fmt.Println(" wallet")
	fmt.Println(" walletlist")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.usage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) Run() {
	cli.validateArgs()

	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	balanceCmd := flag.NewFlagSet("balance", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("wallet", flag.ExitOnError)
	listaddrsCmd := flag.NewFlagSet("walletlist", flag.ExitOnError)

	createAddress := createCmd.String("a", "", "Address to create blockchain")
	balanceAddress := balanceCmd.String("a", "", "Address to get balance from blockchain")
	sendFrom := sendCmd.String("from", "", "Address of sender")
	sendTo := sendCmd.String("to", "", "Address of receiver")
	sendAmount := sendCmd.Int("amount", 0, "Amount sent")

	switch os.Args[1] {
	case "create":
		err := createCmd.Parse(os.Args[2:])
		blockchain.HandleFatalErrors(err)
	case "balance":
		err := balanceCmd.Parse(os.Args[2:])
		blockchain.HandleFatalErrors(err)
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		blockchain.HandleFatalErrors(err)
	case "list":
		err := listCmd.Parse(os.Args[2:])
		blockchain.HandleFatalErrors(err)
	case "wallet":
		err := createWalletCmd.Parse(os.Args[2:])
		blockchain.HandleFatalErrors(err)
	case "walletlist":
		err := listaddrsCmd.Parse(os.Args[2:])
		blockchain.HandleFatalErrors(err)
	default:
		cli.usage()
		runtime.Goexit()
	}

	if createCmd.Parsed() {
		if *createAddress == "" {
			createCmd.Usage()
			runtime.Goexit()
		}
		cli.create(*createAddress)
	}

	if balanceCmd.Parsed() {
		if *balanceAddress == "" {
			balanceCmd.Usage()
			runtime.Goexit()
		}
		cli.balance(*balanceAddress)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}

	if listCmd.Parsed() {
		cli.list()
	}

	if createWalletCmd.Parsed() {
		cli.createWallet()
	}

	if listaddrsCmd.Parsed() {
		cli.listAllAddresses()
	}
}
