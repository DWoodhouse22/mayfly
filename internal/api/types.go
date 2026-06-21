package api

type RegisterRequest struct {
	PublicKey string `json:"public_key"`
	Token     string `json:"token"`
}

type RegisterResponse struct {
	ServerPublicKey string `json:"server_public_key"`
	ClientIP        string `json:"client_ip"`
	DNS             string `json:"dns"`
}
