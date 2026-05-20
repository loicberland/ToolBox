package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"toolBox/modules/v10-lab/internal/lab"
	"toolBox/pkg/modulecontract"

	"github.com/spf13/cobra"
)

var jsonOutput bool

func main() {
	rootCmd := &cobra.Command{
		Use:   "v10-lab",
		Short: "V10 Lab - Generateur de maquettes V10",
	}
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "print JSON output")

	rootCmd.AddCommand(infoCommand())
	rootCmd.AddCommand(productsCommand())
	rootCmd.AddCommand(actionsCommand())
	rootCmd.AddCommand(validateCommand())
	rootCmd.AddCommand(runCommand())
	rootCmd.AddCommand(registerCommand())
	rootCmd.AddCommand(listCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func infoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Print module information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printValue(lab.Info())
		},
	}
}

func productsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "products",
		Short: "Liste les produits supportes",
		RunE: func(cmd *cobra.Command, args []string) error {
			products := lab.Products()
			if jsonOutput {
				return printValue(products)
			}
			for _, product := range products {
				fmt.Fprintf(cmd.OutOrStdout(), "%s - %s\n", product.ID, product.Label)
			}
			return nil
		},
	}
}

func actionsCommand() *cobra.Command {
	var product string
	command := &cobra.Command{
		Use:   "actions",
		Short: "Liste les actions disponibles",
		RunE: func(cmd *cobra.Command, args []string) error {
			actions := lab.Actions()
			if product != "" {
				if !lab.ProductExists(product) {
					return fmt.Errorf("produit inconnu %q", product)
				}
				actions = lab.ActionsForProduct(product)
			}
			if jsonOutput {
				return printValue(actions)
			}
			if product != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Actions disponibles pour %s:\n\n", product)
			}
			for _, action := range actions {
				fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s - %s\n", action.Kind, action.ID, action.Label)
			}
			return nil
		},
	}
	command.Flags().StringVar(&product, "product", "", "filter by product")
	return command
}

func validateCommand() *cobra.Command {
	var configPath string
	command := &cobra.Command{
		Use:   "validate",
		Short: "Valide une configuration JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := lab.LoadConfig(configPath)
			if err != nil {
				return err
			}
			if err := lab.ValidateConfig(config); err != nil {
				if validationErr, ok := err.(lab.ValidationError); ok {
					fmt.Fprintln(cmd.ErrOrStderr(), validationErr.Format())
				}
				return err
			}
			if jsonOutput {
				return printValue(map[string]any{"status": "valid", "name": config.Name, "product": config.Product})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Configuration valide: %s (%s)\n", config.Name, config.Product)
			return nil
		},
	}
	command.Flags().StringVar(&configPath, "config", "", "path to maquette config JSON")
	_ = command.MarkFlagRequired("config")
	return command
}

func runCommand() *cobra.Command {
	var configPath string
	command := &cobra.Command{
		Use:   "run [action]",
		Short: "Execute fictivement un pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" && len(args) == 1 {
				return printValue(modulecontract.ActionResponse{
					ModuleID: lab.ModuleID,
					ActionID: args[0],
					Status:   "done",
					Output: map[string]any{
						"message": "v10-lab action skeleton executed",
					},
				})
			}
			config, err := lab.LoadConfig(configPath)
			if err != nil {
				return err
			}
			if jsonOutput {
				if err := lab.ValidateConfig(config); err != nil {
					return err
				}
				return printValue(modulecontract.ActionResponse{
					ModuleID: lab.ModuleID,
					ActionID: "run",
					Status:   "done",
					Output: map[string]any{
						"name":          config.Name,
						"product":       config.Product,
						"pipelineSteps": len(config.Pipeline),
					},
				})
			}
			return lab.RunPipeline(context.Background(), config, cmd.OutOrStdout())
		},
	}
	command.Flags().StringVar(&configPath, "config", "", "path to maquette config JSON")
	return command
}

func registerCommand() *cobra.Command {
	var configPath string
	command := &cobra.Command{
		Use:   "register",
		Short: "Enregistre une maquette localement",
		RunE: func(cmd *cobra.Command, args []string) error {
			maquette, err := lab.RegisterConfig(configPath)
			if err != nil {
				return err
			}
			if jsonOutput {
				return printValue(maquette)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Maquette enregistrée: %s\n%s\n", maquette.Name, maquette.Path)
			return nil
		},
	}
	command.Flags().StringVar(&configPath, "config", "", "path to maquette config JSON")
	_ = command.MarkFlagRequired("config")
	return command
}

func listCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Liste les maquettes enregistrées",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := lab.ListMaquettes()
			if err != nil {
				return err
			}
			if jsonOutput {
				return printValue(items)
			}
			if len(items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Aucune maquette enregistrée.")
				return nil
			}
			for _, item := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s - %s\n%s\n", item.Name, item.Product, item.Path)
			}
			return nil
		},
	}
}

func printValue(value any) error {
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
	fmt.Printf("%+v\n", value)
	return nil
}
