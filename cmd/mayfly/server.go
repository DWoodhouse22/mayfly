package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"mayfly/internal/api"
	"mayfly/internal/config"
	"mayfly/internal/docker"
	"mayfly/internal/keygen"
	sshclient "mayfly/internal/ssh"
	"mayfly/internal/tunnel"
)

var (
	flagHost   string
	flagUser   string
	flagKey    string
	flagPort   int
	flagImage  string
	flagToken  string
	flagExport string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "manage the VPN server container",
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "provision and start the VPN server on your VPS",
	RunE:  runServerStart,
}

var serverStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop and remove the VPN server container",
	RunE:  runServerStop,
}

func runServerStart(cmd *cobra.Command, args []string) error {
	ssh, err := preFlightChecks()
	if err != nil {
		return fmt.Errorf("pre-flight checks failed: %w", err)
	}
	defer ssh.Close()

	keys, err := keygen.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generating keypair: %w", err)
	}

	log.Printf("starting server container...")
	if err := docker.Start(ssh, flagImage, flagToken); err != nil {
		return err
	}

	containerIP, err := docker.IP(ssh)
	if err != nil {
		return err
	}

	log.Printf("waiting for server to be ready...")
	apiClient := api.NewClient(ssh, containerIP)
	if err := apiClient.WaitHealthy(30 * time.Second); err != nil {
		return err
	}

	log.Printf("registering client...")
	reg, err := apiClient.Register(keys.PublicKey, flagToken)
	if err != nil {
		return err
	}

	cfg := &config.ClientConfig{
		PrivateKey:      keys.PrivateKey,
		ClientIP:        reg.ClientIP,
		DNS:             reg.DNS,
		ServerPublicKey: reg.ServerPublicKey,
		Endpoint:        fmt.Sprintf("%s:51820", flagHost),
	}

	if flagExport != "" {
		if err := config.WriteClient(flagExport, cfg); err != nil {
			return fmt.Errorf("exporting client config: %w", err)
		}
		fmt.Printf("Exported client config: %s\n", flagExport)
	}

	dev, err := tunnel.Up(cfg)
	if err != nil {
		return err
	}

	fmt.Printf("\nVPN server is running.\nPress Ctrl+C to disconnect.\n\n")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\nDisconnecting...")
	if err := tunnel.Down(dev); err != nil {
		return err
	}
	return docker.Stop(ssh)
}

func preFlightChecks() (*sshclient.Client, error) {
	if err := confirmToken(); err != nil {
		return nil, err
	}

	log.Printf("connecting to %s@%s:%d...", flagUser, flagHost, flagPort)
	ssh, err := sshclient.Connect(flagHost, flagUser, flagPort, flagKey)
	if err != nil {
		return nil, fmt.Errorf("SSH connection failed: %w", err)
	}
	return ssh, nil
}

func confirmToken() error {
	if flagToken == "" {
		flagToken = os.Getenv("MAYFLY_TOKEN")
	}
	if flagToken == "" {
		return fmt.Errorf("--token is required (or set MAYFLY_TOKEN)")
	}
	return nil
}

func runServerStop(cmd *cobra.Command, args []string) error {
	log.Printf("connecting to %s@%s:%d...", flagUser, flagHost, flagPort)
	ssh, err := sshclient.Connect(flagHost, flagUser, flagPort, flagKey)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	defer ssh.Close()

	if err := docker.Stop(ssh); err != nil {
		return err
	}

	fmt.Println("Server stopped.")
	return nil
}

func init() {
	serverCmd.AddCommand(serverStartCmd)
	serverCmd.AddCommand(serverStopCmd)

	// SSH flags are shared by all server subcommands.
	serverCmd.PersistentFlags().StringVar(&flagHost, "host", "", "VPS hostname or IP")
	serverCmd.PersistentFlags().StringVarP(&flagUser, "user", "u", "root", "SSH login user")
	serverCmd.PersistentFlags().StringVarP(&flagKey, "key", "k", "", "SSH private key file")
	serverCmd.PersistentFlags().IntVarP(&flagPort, "port", "p", 22, "SSH port")
	serverCmd.MarkPersistentFlagRequired("host")

	serverStartCmd.Flags().StringVarP(&flagImage, "image", "i", "ghcr.io/dwoodhouse22/mayfly-server:latest", "server Docker image")
	serverStartCmd.Flags().StringVarP(&flagToken, "token", "t", "", "auth token (or set MAYFLY_TOKEN)")
	serverStartCmd.Flags().StringVarP(&flagExport, "export", "e", "", "export the client WireGuard config to this path")
}
