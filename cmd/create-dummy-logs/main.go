package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"
)

var (
	subdirectory = "logs"

	daemonNames = []string{
		"watcher",
		"worker",
		"processor",
		"updater",
	}
)

func main() {
	for _, daemonName := range daemonNames {
		directory := filepath(subdirectory, daemonName)

		if err := os.MkdirAll(directory, 0755); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"error creating directory (%s): %s\n",
				daemonName,
				err.Error(),
			)

			os.Exit(1)
		}

		if err := writeFiles(directory); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"error writing files (%s): %s\n",
				daemonName,
				err.Error(),
			)

			os.Exit(1)
		}
	}
}

func writeFiles(directory string) error {
	slogTxt := createTextLogs()

	err := os.WriteFile(filepath(directory, "slog.txt"), []byte(slogTxt), 0644)
	if err != nil {
		return fmt.Errorf("error writing slog.txt: %w", err)
	}

	slogJSON := createJSONLogs()

	err = os.WriteFile(filepath(directory, "slog.json"), []byte(slogJSON), 0644)
	if err != nil {
		return fmt.Errorf("error writing slog.json: %w", err)
	}

	return nil
}

func createJSONLogs() string {
	var slogJSON strings.Builder

	logger := slog.New(slog.NewJSONHandler(&slogJSON, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	createSlogLogs(logger)

	return slogJSON.String()
}

func createTextLogs() string {
	var slogTxt strings.Builder

	logger := slog.New(slog.NewTextHandler(&slogTxt, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	createSlogLogs(logger)

	return slogTxt.String()
}

func createSlogLogs(logger *slog.Logger) {
	logger.Warn("This is a warning message", slog.Any("fields", map[string]string{"key": "value"}))
	logger.Info("This is an info message", slog.String("key", "value"))
	logger.Debug("This is a debug message", slog.Time("time", time.Now()))
}

func filepath(directory, filename string) string {
	return path.Join(directory, filename)
}
