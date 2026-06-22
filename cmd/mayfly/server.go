package main

import (
	"fmt"
	"log"
	sshclient "mayfly/internal/ssh"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagHost  string
	flagUser  string
	flagKey   string
	flagPort  int
	flagImage string
	flagToken string
)

var serverCmd = &cobra.Command{
	Use: "server",
}

var serverStartCmd = &cobra.Command{
	Use:  "start",
	RunE: runServerStart,
}
var serverStopCmd = &cobra.Command{
	Use:  "stop",
	RunE: runServerStop,
}

func runServerStart(cmd *cobra.Command, args []string) error {
	if flagToken == "" {
		flagToken = os.Getenv("MAYFLY_TOKEN")
	}
	if flagToken == "" {
		return fmt.Errorf("--token is required (or set MAYFLY_TOKEN)")
	}

	fmt.Printf("Connecting to %s@%s:%d...\n", flagUser, flagHost, flagPort)

	client, err := sshclient.Connect(flagHost, flagUser, flagPort, flagKey)
	if err != nil {
		log.Fatalf("SSH connection failed: %v", err)
	}
	defer client.Close()

	out, err := client.Run("uname -sr")
	if err != nil {
		log.Fatalf("command failed: %v", err)
	}

	fmt.Printf("Connected. Remote is running: %s\n", out)

	return nil
}

func runServerStop(cmd *cobra.Command, args []string) error {
	return nil
}

func init() {
	serverCmd.AddCommand(serverStartCmd)
	serverCmd.AddCommand(serverStopCmd)

	serverStartCmd.Flags().StringVar(&flagHost, "host", "", "Hostname of the VPS")
	serverStartCmd.Flags().StringVarP(&flagUser, "user", "u", "", "SSH login user")
	serverStartCmd.Flags().StringVarP(&flagKey, "key", "k", "", "Path to SSH private key file")
	serverStartCmd.Flags().IntVarP(&flagPort, "port", "p", 22, "SSH port")
	serverStartCmd.Flags().StringVarP(&flagImage, "image", "i", "ghcr.io/DWoodhouse22/mayfly-server:latest", "Server docker image")
	serverStartCmd.Flags().StringVarP(&flagToken, "token", "t", "", "Access token")

	serverStartCmd.MarkFlagRequired("host")
}
