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
