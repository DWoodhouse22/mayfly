package main

import "github.com/spf13/cobra"

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
	return nil
}

func runServerStop(cmd *cobra.Command, args []string) error {
	return nil
}
