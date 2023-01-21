package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func NewWallet() *Wallet {
	pvtkey, pubkey := NewKeyPair()
	return &Wallet{pvtkey, pubkey}
}

func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()

	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	HandleFatalErrors(err)

	public := append(private.PublicKey.X.Bytes(), private.Y.Bytes()...)

	return *private, public
}

func (w Wallet) Address() []byte {
	pubHash := PublicKeyHash(w.PublicKey)
	verHash := append([]byte{version}, pubHash...)
	checksum := Checksum(verHash)

	hash := append(verHash, checksum...)
	address := Base58Encode(hash)

	// fmt.Printf("public key: %x\n", w.PublicKey)
	// fmt.Printf("public key hash: %x\n", pubHash)
	// fmt.Printf("address: %s\n", address)

	return address
}
