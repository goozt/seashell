package blockchain

import (
	"bytes"
	"encoding/binary"
	"log"
	"math/big"
)

func HandleFatalErrors(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func ToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	HandleFatalErrors(err)
	return buff.Bytes()
}

// encodeSig encodes ECDSA (r, s) as a fixed 64-byte value (32 bytes each, big-endian zero-padded).
// This ensures correct round-trip even when r or s has fewer than 32 significant bytes.
func encodeSig(r, s *big.Int) []byte {
	sig := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):64], sBytes)
	return sig
}

// decodeSig splits a fixed 64-byte encoded signature back into (r, s).
func decodeSig(sig []byte) (r, s big.Int) {
	r.SetBytes(sig[:32])
	s.SetBytes(sig[32:])
	return r, s
}

// encodePubKey encodes an ECDSA public key (x, y) as a fixed 64-byte value.
func encodePubKey(x, y *big.Int) []byte {
	pub := make([]byte, 64)
	xBytes := x.Bytes()
	yBytes := y.Bytes()
	copy(pub[32-len(xBytes):32], xBytes)
	copy(pub[64-len(yBytes):64], yBytes)
	return pub
}

// decodePubKey splits a fixed 64-byte encoded public key back into (x, y).
func decodePubKey(pub []byte) (x, y big.Int) {
	x.SetBytes(pub[:32])
	y.SetBytes(pub[32:])
	return x, y
}

// SplitBinary splits data at midpoint into two big.Ints.
// Deprecated for signatures: use encodeSig/decodeSig which handle variable-length r/s correctly.
func SplitBinary(data []byte) (a big.Int, b big.Int) {
	length := len(data)
	a.SetBytes(data[:(length / 2)])
	b.SetBytes(data[(length / 2):])
	return a, b
}
