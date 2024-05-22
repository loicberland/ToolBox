/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var command1Cmd = &cobra.Command{
	Use:   "command1",
	Short: "A brief description of your command",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("command1 called")
	},
}

func init() {
	rootCmd.AddCommand(command1Cmd)
}
