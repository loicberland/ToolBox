package cmd

import (
	namechanger "toolBox/pkg/nameChanger"
	"toolBox/pkg/utils"

	"github.com/spf13/cobra"
)

var argOption int
var argDirectory string
var argFind string
var argReplace string

var nameChangerCmd = &cobra.Command{
	Use:   "name-changer",
	Short: "Effortlessly search and replace strings in directory and file names.",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		namechanger.Namechanger(argOption, argDirectory, argFind, argReplace)
	},
}

func init() {
	othersCmd.AddCommand(nameChangerCmd)
	nameChangerCmd.PersistentFlags().IntVarP(&argOption, "option", "c", 1, "1=files/folders, 2=folders, 3=files")
	nameChangerCmd.PersistentFlags().StringVarP(&argDirectory, "path", "p", utils.GetCurrentDirectory(), "directory")
	nameChangerCmd.PersistentFlags().StringVarP(&argFind, "search", "f", "", "string to search")
	nameChangerCmd.MarkPersistentFlagRequired("search")
	nameChangerCmd.PersistentFlags().StringVarP(&argReplace, "replace", "r", "", "replacement string")
	nameChangerCmd.MarkPersistentFlagRequired("replace")
}
