package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"sike/adapter/store"
	"sike/domain/service"
	"sike/view/http/server"
)

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "start the web server",
		Long: `start the web server. SIKE_ADDR sets the listen address (default :8090),
SIKE_DB the database file (default sike.db), created and migrated on boot.`,
		Example: `  sike run
  SIKE_ADDR=:3000 SIKE_DB=/data/sike.db sike run`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, path, v, err := openDB(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			log.Printf("db %s at version %d", path, v)

			st := store.New(db)
			srv, err := server.New(server.Config{
				Addr:  getenv("SIKE_ADDR", ":8090"),
				Users: service.NewUsers(st),
				Areas: service.NewAreas(st),
				Tasks: service.NewTasks(st),
			})
			if err != nil {
				return err
			}
			return srv.Run(cmd.Context())
		},
	}
}
