package blockchain

import (
	"bytes"
	"encoding/binary"
	"log"
)

func HandleFatalErrors(err error) {
	if err != nil {
		log.Panic(err)
		// log.Fatalln(err)
	}
}

func ToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	HandleFatalErrors(err)
	return buff.Bytes()
}
