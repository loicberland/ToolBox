package main

import (
	"encoding/json"
	"fmt"
	"os"

	"toolBox/modules/test-env/internal/actions"
	"toolBox/modules/test-env/internal/config"

	"github.com/spf13/cobra"
)

var jsonOutput bool

func main() {
	rootCmd := &cobra.Command{
		Use:   "test-env",
		Short: "Test environment module",
	}
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "print JSON output")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "info",
		Short: "Print module information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return print(actions.Info())
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "actions",
		Short: "Print available actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return print(actions.Actions())
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "run <action>",
		Short: "Run an action",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "init-config" {
				if err := config.Ensure(""); err != nil {
					return err
				}
			}
			return print(actions.Run(args[0]))
		},
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func print(value any) error {
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
	fmt.Printf("%+v\n", value)
	return nil
}
