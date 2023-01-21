package wallet

import (
	"bytes"
	"encoding/gob"
	"os"

	"crypto/elliptic"
)

const walletFile = "./db/wallets.data"

type WalletDB struct {
	Wallets map[string]*Wallet
}

func CreateWalletDB() (*WalletDB, error) {

	if _, err := os.Stat("./db"); os.IsNotExist(err) {
		_ = os.MkdirAll("./db", 0700)
	}

	walletDB := WalletDB{}
	walletDB.Wallets = make(map[string]*Wallet)

	err := walletDB.LoadFile()

	return &walletDB, err
}

func (wdb WalletDB) GetWallet(address string) Wallet {
	return *wdb.Wallets[address]
}

func (wdb WalletDB) GetAllWallet() []string {
	var addresses []string

	for address := range wdb.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

func (wdb *WalletDB) AddWallet() string {
	wallet := NewWallet()
	address := string(wallet.Address())

	wdb.Wallets[address] = wallet

	return address
}

func (wdb *WalletDB) LoadFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	var walletDB WalletDB

	fileContent, err := os.ReadFile(walletFile)
	HandleFatalErrors(err)

	gob.Register(elliptic.P256())

	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&walletDB)
	HandleFatalErrors(err)

	wdb.Wallets = walletDB.Wallets

	return nil
}

func (wdb *WalletDB) SaveFile() {
	var content bytes.Buffer

	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&content)

	err := encoder.Encode(wdb)
	HandleFatalErrors(err)

	err = os.WriteFile(walletFile, content.Bytes(), 0644)
	HandleFatalErrors(err)
}
