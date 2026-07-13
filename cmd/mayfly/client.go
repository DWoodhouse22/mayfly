package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"mayfly/internal/api"
	"mayfly/internal/config"
	"mayfly/internal/docker"
	"mayfly/internal/keygen"
	sshclient "mayfly/internal/ssh"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "manage VPN clients",
}

var clientJoinCmd = &cobra.Command{
	Use:   "join",
	Short: "connect a client to the VPN",
	RunE:  runClientJoin,
}

func runClientJoin(cmd *cobra.Command, args []string) error {
	log.Printf("connecting to %s@%s:%d...", flagUser, flagHost, flagPort)
	ssh, err := sshclient.Connect(flagHost, flagUser, flagPort, flagKey)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	defer ssh.Close()

	containerIP, err := docker.IP(ssh)
	if err != nil {
		return fmt.Errorf("no running mayfly server found on %s - start one first with `mayfly server start`: %w", flagHost, err)
	}

	keys, err := keygen.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generating keypair: %w", err)
	}

	log.Printf("registering client...")
	apiClient := api.NewClient(ssh, containerIP)
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

	if err := config.WriteClient(flagExport, cfg); err != nil {
		return fmt.Errorf("exporting client config: %w", err)
	}

	fmt.Printf("Exported client config: %s\n", flagExport)
	fmt.Println("Transfer this file to your device and import it into the WireGuard app.")
	return nil
}

func init() {
	clientCmd.AddCommand(clientJoinCmd)
	clientCmd.PersistentFlags().StringVar(&flagHost, "host", "", "VPS hostname or IP")
	clientCmd.PersistentFlags().StringVarP(&flagUser, "user", "u", "root", "SSH login user")
	clientCmd.PersistentFlags().StringVarP(&flagKey, "key", "k", "", "SSH private key file")
	clientCmd.PersistentFlags().IntVarP(&flagPort, "port", "p", 22, "SSH port")
	clientCmd.MarkPersistentFlagRequired("host")

	clientJoinCmd.Flags().StringVarP(&flagToken, "token", "t", "", "auth token (must match the server's token)")
	clientJoinCmd.Flags().StringVarP(&flagExport, "export", "e", "", "write the client WireGuard config to this path")
	clientJoinCmd.MarkFlagRequired("token")
	clientJoinCmd.MarkFlagRequired("export")
}
