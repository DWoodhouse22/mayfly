package main

import (
	"encoding/json"
	"log"
	"mayfly/internal/api"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const dnsIP = "10.0.0.1"

type server struct {
	wg    *wgServer
	token string
}

func main() {
	token := os.Getenv("MAYFLY_TOKEN")
	if token == "" {
		log.Fatal("MAYFLY_TOKEN not found")
	}

	wgServer, err := newWGServer()
	if err != nil {
		log.Fatalf("failed to create new wgServer: %v", err)
	}
	defer wgServer.Close()

	unboundCmd, err := startUnbound()
	if err != nil {
		log.Default().Fatalf("failed to start unbound: %v", err)
	}
	defer unboundCmd.Process.Kill()

	s := &server{
		wg:    wgServer,
		token: token,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/register", s.handleRegister)

	go func() {
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatalf("failed serving http: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down")
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleRegister(w http.ResponseWriter, r *http.Request) {
	req := &api.RegisterRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.Token != s.token {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	pubKey, err := wgtypes.ParseKey(req.PublicKey)
	if err != nil {
		http.Error(w, "failed to parse public key", http.StatusBadRequest)
		return
	}

	clientIP := "10.0.0.2"
	if err := s.wg.AddPeer(pubKey, net.ParseIP(clientIP)); err != nil {
		http.Error(w, "failed to add peer", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api.RegisterResponse{
		ServerPublicKey: s.wg.PublicKey.String(),
		ClientIP:        clientIP,
		DNS:             dnsIP,
	})
}
