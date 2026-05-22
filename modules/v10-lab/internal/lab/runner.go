package lab

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func RunPipeline(ctx context.Context, config Config, writer io.Writer) error {
	ApplyDefaults(&config)
	logWriter, closeLog, logPath, err := runLogWriter(config, writer)
	if err != nil {
		return err
	}
	defer closeLog()

	if logPath != "" {
		fmt.Fprintf(logWriter, "Log: %s\n\n", logPath)
	}
	if err := ValidateConfig(config); err != nil {
		if validationErr, ok := err.(ValidationError); ok {
			fmt.Fprintln(logWriter, validationErr.Format())
		}
		return err
	}
	fmt.Fprintf(logWriter, "V10 Lab - Exécution maquette %s\n", config.Name)
	fmt.Fprintf(logWriter, "Produit: %s\n", config.Product)
	fmt.Fprintf(logWriter, "Cible: %s\n\n", ResolveMaquetteTargetPath(config))
	for index, step := range config.Pipeline {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		action, _ := FindAction(step.Action)
		label := step.Label
		if strings.TrimSpace(label) == "" {
			label = action.Label
		}
		params := paramsWithDefaults(action, step.Params)
		fmt.Fprintf(logWriter, "[%d/%d] %s - %s\n", index+1, len(config.Pipeline), action.ID, label)
		if err := action.Execute(ActionContext{Writer: logWriter, Config: config, Step: step}, params); err != nil {
			fmt.Fprintf(logWriter, "Erreur: %v\n", err)
			return err
		}
		fmt.Fprintln(logWriter)
	}
	fmt.Fprintln(logWriter, "Exécution terminée.")
	return nil
}

func RunAction(ctx context.Context, config Config, actionID string, writer io.Writer) error {
	ApplyDefaults(&config)
	action, ok := FindAction(actionID)
	if !ok {
		return fmt.Errorf("action inconnue %q", actionID)
	}
	if !action.SupportsProduct(config.Product) {
		return fmt.Errorf("action %q incompatible avec le produit %q", actionID, config.Product)
	}
	config.Pipeline = []PipelineStep{{Action: actionID, Label: action.Label, Params: map[string]any{}}}

	logWriter, closeLog, logPath, err := runLogWriter(config, writer)
	if err != nil {
		return err
	}
	defer closeLog()

	if logPath != "" {
		fmt.Fprintf(logWriter, "Log: %s\n\n", logPath)
	}
	if err := ValidateConfig(config); err != nil {
		if validationErr, ok := err.(ValidationError); ok {
			fmt.Fprintln(logWriter, validationErr.Format())
		}
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	fmt.Fprintf(logWriter, "V10 Lab - Action maquette %s\n", config.Name)
	fmt.Fprintf(logWriter, "Produit: %s\n", config.Product)
	fmt.Fprintf(logWriter, "Cible: %s\n\n", ResolveMaquetteTargetPath(config))
	fmt.Fprintf(logWriter, "[1/1] %s - %s\n", action.ID, action.Label)
	if err := action.Execute(ActionContext{Writer: logWriter, Config: config, Step: config.Pipeline[0]}, paramsWithDefaults(action, map[string]any{})); err != nil {
		fmt.Fprintf(logWriter, "Erreur: %v\n", err)
		return err
	}
	fmt.Fprintln(logWriter)
	fmt.Fprintln(logWriter, "ExÃ©cution terminÃ©e.")
	return nil
}

func runLogWriter(config Config, writer io.Writer) (io.Writer, func(), string, error) {
	if writer == nil {
		writer = io.Discard
	}
	logDir := filepath.Join(MaquettesDir(), safeDirName(config.Name), "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, func() {}, "", err
	}
	logPath := filepath.Join(logDir, time.Now().Format("20060102-150405")+"-run.log")
	file, err := os.Create(logPath)
	if err != nil {
		return nil, func() {}, "", err
	}
	return io.MultiWriter(writer, file), func() { _ = file.Close() }, logPath, nil
}
