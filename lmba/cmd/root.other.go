/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var othersCmd = &cobra.Command{
	Use:   "others",
	Short: "List of Various Tools",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			log.Fatalf("unable to print help: %s\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(othersCmd)
}
