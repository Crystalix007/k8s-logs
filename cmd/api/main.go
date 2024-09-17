package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/crystalix007/log-viewer/api"
	"github.com/spf13/cobra"
)

// Flags represents the command-line flags for the program.
type Flags struct {
	Address          *string
	WorkingDirectory *string
}

func main() {
	var flags Flags

	cmd := cobra.Command{
		Use:   "api",
		Short: "Start the log viewing service API",
		RunE: func(cmd *cobra.Command, args []string) error {
			return serve(flags)
		},
	}

	flags.Address = cmd.Flags().StringP("address", "a", "localhost:0", "the address to listen on")
	flags.WorkingDirectory = cmd.Flags().
		StringP("working-directory", "w", "", "the working directory for the API")

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func serve(flags Flags) error {
	var apiOpts []api.Option

	if *flags.WorkingDirectory != "" {
		apiOpts = append(
			apiOpts,
			api.WithWorkingDirectory(*flags.WorkingDirectory),
		)

		slog.Info(
			"Working directory set",
			slog.String("working_directory", *flags.WorkingDirectory),
		)
	}

	api, err := api.New(apiOpts...)
	if err != nil {
		panic(err)
	}

	listener, err := net.Listen("tcp", *flags.Address)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Listening on http://%s\n", listener.Addr().String())

	defer listener.Close()

	if err := http.Serve(listener, api); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}

	return nil
}
