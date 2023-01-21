package wallet

import (
	"bytes"
	"log"

	"crypto/sha256"

	"github.com/mr-tron/base58"
	"golang.org/x/crypto/sha3"
)

const (
	ChecksumLength = 4
	version        = byte(0x00)
)

func PublicKeyHash(pubKey []byte) []byte {
	pubHash := sha256.Sum256(pubKey)

	publicKeyHash := make([]byte, 20)
	sha3.ShakeSum256(publicKeyHash, pubHash[:])

	return publicKeyHash
}

func Checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:ChecksumLength]
}

func Base58Encode(input []byte) []byte {
	encoded := base58.Encode(input)
	return []byte(encoded)
}

func Base58Decode(input []byte) []byte {
	decoded, err := base58.Decode(string(input[:]))
	if err != nil {
		log.Fatalln(err)
	}
	return decoded
}

func ValidateAddress(address string) bool {
	pubHashKey := Base58Decode([]byte(address))
	checksumIdx := len(pubHashKey) - ChecksumLength
	actualChecksum := pubHashKey[checksumIdx:]
	version := pubHashKey[0]
	pubHashKey = pubHashKey[1:checksumIdx]
	targetChecksum := Checksum(append([]byte{version}, pubHashKey...))

	return bytes.Equal(actualChecksum, targetChecksum)
}
