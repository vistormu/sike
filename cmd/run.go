package cmd

import (
	"log"
	"time"

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
SIKE_DB the database file (default sike.db), created and migrated on boot,
SIKE_TZ the timezone times are in (default Europe/Madrid).`,
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

			loc, err := location()
			if err != nil {
				return err
			}
			st := store.New(db)
			users, tasks := service.NewUsers(st), service.NewTasks(st)
			notifier := service.NewNotifier(notifierPort(), st, users, tasks, loc)
			srv, err := server.New(server.Config{
				Addr:     getenv("SIKE_ADDR", ":8090"),
				Users:    users,
				Areas:    service.NewAreas(st),
				Tasks:    tasks,
				Moods:    service.NewMoods(st),
				Notifier: notifier,
				Ping:     db.PingContext,
				Loc:      loc,
				Now:      time.Now,
			})
			if err != nil {
				return err
			}
			// the minute ticker, in the same binary (technical-design.md)
			if notifierPort() != nil {
				log.Printf("notifications to %s", getenv("SIKE_NTFY_URL", ""))
				go notifier.Run(cmd.Context(), time.Now)
			} else {
				log.Printf("notifications off: no SIKE_NTFY_URL")
			}
			return srv.Run(cmd.Context())
		},
	}
}
