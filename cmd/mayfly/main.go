package main

import (
	"fmt"
	"log"

	"mayfly/internal/keygen"
)

func main() {
	server, err := keygen.GenerateKeyPair()
	if err != nil {
		log.Fatalf("generate server keypair: %v", err)
	}

	client, err := keygen.GenerateKeyPair()
	if err != nil {
		log.Fatalf("generate client keypair: %v", err)
	}

	fmt.Println("Server:")
	fmt.Printf("private: %s\n", server.PrivateKey)
	fmt.Printf("public:  %s\n", server.PublicKey)
	fmt.Println("Client:")
	fmt.Printf("private: %s\n", client.PrivateKey)
	fmt.Printf("public:  %s\n", client.PublicKey)
}
