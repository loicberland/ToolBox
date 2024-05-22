/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var argNoWait bool
var version = "0.0.1"

var rootCmd = &cobra.Command{
	Use:     "perso-toolBox",
	Short:   "A toolbox for perso that combines several useful everyday features.",
	Long:    ``,
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("This is the first cobra example")
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	//disable help command
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
	//disable completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().BoolVar(&argNoWait, "t", false, "Help message for toggle")
}
