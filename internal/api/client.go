package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	sshclient "mayfly/internal/ssh"
)

// Client talks to the server API running inside the container.
// All HTTP traffic is carried over the SSH tunnel, the container's port 8080 is never exposed on the public network.
type Client struct {
	http *http.Client
}

func NewClient(ssh *sshclient.Client, containerIP string) *Client {
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return ssh.Dial(containerIP + ":8080")
		},
	}
	return &Client{
		http: &http.Client{
			Transport: transport,
			Timeout:   5 * time.Second,
		},
	}
}

func (c *Client) WaitHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := c.http.Get("http://mayfly-server/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for server to become ready")
}

func (c *Client) Register(clientPubKey, token string) (*RegisterResponse, error) {
	body, err := json.Marshal(RegisterRequest{
		PublicKey: clientPubKey,
		Token:     token,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Post("http://mayfly-server/register", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("register request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("register failed with status %s", resp.Status)
	}

	var reg RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&reg); err != nil {
		return nil, fmt.Errorf("decoding register response: %w", err)
	}
	return &reg, nil
}
