package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"

	"mayfly/internal/api"
	"mayfly/internal/config"
	"mayfly/internal/docker"
	"mayfly/internal/keygen"
	sshclient "mayfly/internal/ssh"
)

var (
	flagHost   string
	flagUser   string
	flagKey    string
	flagPort   int
	flagImage  string
	flagToken  string
	flagOutput string
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
	if flagToken == "" {
		flagToken = os.Getenv("MAYFLY_TOKEN")
	}
	if flagToken == "" {
		return fmt.Errorf("--token is required (or set MAYFLY_TOKEN)")
	}

	keys, err := keygen.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generating keypair: %w", err)
	}

	log.Printf("connecting to %s@%s:%d...", flagUser, flagHost, flagPort)
	ssh, err := sshclient.Connect(flagHost, flagUser, flagPort, flagKey)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	defer ssh.Close()

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

	cfg := config.ClientConfig{
		PrivateKey:      keys.PrivateKey,
		ClientIP:        reg.ClientIP,
		DNS:             reg.DNS,
		ServerPublicKey: reg.ServerPublicKey,
		Endpoint:        fmt.Sprintf("%s:51820", flagHost),
	}
	if err := config.WriteClient(flagOutput, cfg); err != nil {
		return err
	}

	fmt.Printf("\nVPN server is running.\n")
	fmt.Printf("Client config written to: %s\n\n", flagOutput)
	fmt.Printf("Connect with:\n  sudo wg-quick up %s\n\n", flagOutput)
	fmt.Printf("When done:\n  sudo wg-quick down %s\n  mayfly server stop --host %s\n", flagOutput, flagHost)

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
	serverStartCmd.Flags().StringVarP(&flagOutput, "output", "o", "mayfly.conf", "path to write the WireGuard client config")
}
