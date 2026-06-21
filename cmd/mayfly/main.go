package main

import (
	"flag"
	"fmt"
	"log"

	"mayfly/internal/keygen"
	sshclient "mayfly/internal/ssh"
)

func main() {
	host := flag.String("host", "", "VPS host to connect to (user@host)")
	user := flag.String("user", "root", "SSH user")
	key := flag.String("key", "", "SSH private key file (default: ~/.ssh/id_ed25519)")
	port := flag.Int("port", 22, "SSH port")
	flag.Parse()

	if *host != "" {
		testSSH(*host, *user, *key, *port)
		return
	}

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

func testSSH(host, user, key string, port int) {
	fmt.Printf("Connecting to %s@%s:%d...\n", user, host, port)

	client, err := sshclient.Connect(host, user, port, key)
	if err != nil {
		log.Fatalf("SSH connection failed: %v", err)
	}
	defer client.Close()

	out, err := client.Run("uname -sr")
	if err != nil {
		log.Fatalf("command failed: %v", err)
	}

	fmt.Printf("Connected. Remote is running: %s\n", out)
}
