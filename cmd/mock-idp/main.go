// cmd/mock-idp is a lightweight mock Identity Provider server for development.
//
// GET /verify?id=<12-digit-national-id>
//   → 200 "Verified User <last4>"   (valid 12-digit input)
//   → 400 "invalid national ID"     (wrong format)
//
// Usage:
//
//	go run ./cmd/mock-idp -port 9999
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"regexp"
)

var nationalIDPattern = regexp.MustCompile(`^\d{12}$`)

func main() {
	port := flag.Int("port", 9999, "Port to listen on")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if !nationalIDPattern.MatchString(id) {
			http.Error(w, "invalid national ID", http.StatusBadRequest)
			return
		}
		last4 := id[len(id)-4:]
		fmt.Fprintf(w, "Verified User %s", last4)
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("mock-idp listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
