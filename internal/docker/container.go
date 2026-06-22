package docker

import (
	"fmt"
	"strings"

	sshclient "mayfly/internal/ssh"
)

const containerName = "mayfly-server"

func Start(client *sshclient.Client, image, token string) error {
	client.Run(fmt.Sprintf("docker rm -f %s", containerName))

	cmd := fmt.Sprintf(
		"docker run -d --name %s --cap-add NET_ADMIN --device /dev/net/tun --sysctl net.ipv4.ip_forward=1 -p 51820:51820/udp -e MAYFLY_TOKEN=%s %s",
		containerName,
		shellQuote(token),
		image,
	)
	out, err := client.Run(cmd)
	if err != nil {
		return fmt.Errorf("starting container: %w\n%s", err, out)
	}
	return nil
}

// IP returns the Docker bridge IP of the running container, used to reach its HTTP API through the SSH tunnel.
func IP(client *sshclient.Client) (string, error) {
	out, err := client.Run(fmt.Sprintf(
		"docker inspect --format '{{.NetworkSettings.IPAddress}}' %s",
		containerName,
	))
	if err != nil {
		return "", fmt.Errorf("error inspecting container: %w", err)
	}
	ip := strings.TrimSpace(out)
	if ip == "" {
		return "", fmt.Errorf("container %s has no IP address", containerName)
	}
	return ip, nil
}

func Stop(client *sshclient.Client) error {
	out, err := client.Run(fmt.Sprintf("docker rm -f %s", containerName))
	if err != nil {
		return fmt.Errorf("stopping container: %w\n%s", err, out)
	}
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
