//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/styx-oracle/styx/api"
)

func main() {
	port := "8080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	selfID := uint64(1)
	server := api.NewServer(selfID)

	addr := ":" + port
	fmt.Printf("styx oracle listening on %s\n", addr)
	fmt.Println("endpoints:")
	fmt.Println("  GET  /health          - health check")
	fmt.Println("  GET  /query?target=ID - query node status")
	fmt.Println("  POST /report          - submit witness report")
	fmt.Println("  POST /witnesses       - register witness")

	if err := server.ListenAndServe(addr); err != nil {
		log.Fatal(err)
	}
}
