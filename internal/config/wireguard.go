package config

import (
	"fmt"
	"os"
	"text/template"
)

const clientConfigTemplate = `[Interface]
PrivateKey = {{ .PrivateKey }}
Address = {{ .ClientIP }}/24
DNS = {{ .DNS }}

[Peer]
PublicKey = {{ .ServerPublicKey }}
Endpoint = {{ .Endpoint }}
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
`

type ClientConfig struct {
	PrivateKey      string
	ClientIP        string
	DNS             string
	ServerPublicKey string
	Endpoint        string // host:port
}

func WriteClient(path string, cfg *ClientConfig) error {
	tmpl, err := template.New("wg").Parse(clientConfigTemplate)
	if err != nil {
		return fmt.Errorf("error parsing config template: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("error creating config file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, cfg); err != nil {
		return fmt.Errorf("error writing config: %w", err)
	}
	return nil
}
