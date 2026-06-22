package ssh

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Client struct {
	conn *ssh.Client
}

func Connect(host, user string, port int, keyPath string) (*Client, error) {
	auths, err := authMethods(keyPath)
	if err != nil {
		return nil, err
	}

	hostKeyCallback, err := hostKeyVerifier()
	if err != nil {
		return nil, err
	}

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: hostKeyCallback,
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", addr, err)
	}

	return &Client{conn: conn}, nil
}

// Dial opens a TCP connection to addr through the SSH tunnel. The returned conn never leaves the encrypted SSH session,
// so addr does not need to be reachable from the public internet.
func (c *Client) Dial(addr string) (net.Conn, error) {
	return c.conn.Dial("tcp", addr)
}

// Run executes a command on the remote host and returns its combined stdout/stderr output.
func (c *Client) Run(cmd string) (string, error) {
	sess, err := c.conn.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()

	out, err := sess.CombinedOutput(cmd)
	return strings.TrimSpace(string(out)), err
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func authMethods(keyPath string) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	// Prefer SSH agent so the key never has to be read from disk.
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if conn, err := net.Dial("unix", sock); err == nil {
			methods = append(methods, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
		}
	}

	candidates := keyPaths(keyPath)
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("parsing SSH key %s: %w", p, err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
		break
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no SSH authentication methods available (tried agent and %v)", candidates)
	}

	return methods, nil
}

// keyPaths returns the key file(s) to try. When an explicit path is given that
// path is returned alone; otherwise the default key names are expanded against
func keyPaths(explicit string) []string {
	if explicit != "" {
		return []string{explicit}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	// defaultKeyNames are tried in order when no explicit key path is given,
	// matching the lookup order of the OpenSSH client.
	defaultKeyNames := []string{"id_ed25519", "id_ecdsa", "id_rsa"}
	paths := make([]string, len(defaultKeyNames))
	for i, name := range defaultKeyNames {
		paths[i] = filepath.Join(home, ".ssh", name)
	}
	return paths
}

// hostKeyVerifier returns a callback that implements trust-on-first-use against
// ~/.ssh/known_hosts, matching the behaviour of the OpenSSH client.
func hostKeyVerifier() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("finding home directory: %w", err)
	}
	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if _, err := os.Stat(knownHostsPath); err == nil {
			cb, err := knownhosts.New(knownHostsPath)
			if err != nil {
				return fmt.Errorf("reading known_hosts: %w", err)
			}

			err = cb(hostname, remote, key)
			if err == nil {
				return nil // known and verified
			}

			var keyErr *knownhosts.KeyError
			if errors.As(err, &keyErr) && len(keyErr.Want) > 0 {
				// The host is known but the key has changed
				return fmt.Errorf("host key mismatch for %s - possible MITM attack. Check %s", hostname, knownHostsPath)
			}
			// Host not in known_hosts yet, fall through to TOFU prompt.
		}

		fingerprint := ssh.FingerprintSHA256(key)
		fmt.Printf("The authenticity of host '%s' can't be established.\n", hostname)
		fmt.Printf("%s key fingerprint is %s\n", key.Type(), fingerprint)
		fmt.Print("Are you sure you want to continue connecting (yes/no)? ")

		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if strings.TrimSpace(strings.ToLower(scanner.Text())) != "yes" {
			return fmt.Errorf("connection rejected")
		}

		f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("saving host key: %w", err)
		}
		defer f.Close()

		_, err = fmt.Fprintln(f, knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key))
		return err
	}, nil
}
