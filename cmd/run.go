package cmd

import (
	"github.com/spf13/cobra"

	"sike/view/http/server"
)

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "start the web server",
		Long:  `start the web server. SIKE_ADDR sets the listen address (default :8090).`,
		Example: `  sike run
  SIKE_ADDR=:3000 sike run`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := server.New(server.Config{
				Addr: getenv("SIKE_ADDR", ":8090"),
			})
			return srv.Run(cmd.Context())
		},
	}
}
