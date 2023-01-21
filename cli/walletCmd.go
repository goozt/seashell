package cli

import (
	"fmt"

	"github.com/goozt/seashell/wallet"
)

func (cli *CommandLine) createWallet() {
	walletDB, _ := wallet.CreateWalletDB()
	address := walletDB.AddWallet()

	walletDB.SaveFile()

	fmt.Printf("New address is %s\n", address)
}

func (cli *CommandLine) listAllAddresses() {
	walletDB, _ := wallet.CreateWalletDB()
	addresses := walletDB.GetAllWallet()

	for _, address := range addresses {
		fmt.Println(address)
	}
}
