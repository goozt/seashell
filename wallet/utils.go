package wallet

import "log"

func HandleFatalErrors(err error) {
	if err != nil {
		log.Panic(err)
		// log.Fatalln(err)
	}
}
