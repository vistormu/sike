package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"sike/adapter/ntfy"
	"sike/domain/port"
	"sike/domain/service"
)

func opNotifyCmd() *cobra.Command {
	notify := &cobra.Command{
		Use:   "notify",
		Short: "notifications: ntfy (SIKE_NTFY_URL, SIKE_NTFY_TOKEN)",
	}

	test := &cobra.Command{
		Use:     "test <user-id>",
		Short:   "send a test message to a person's ntfy topic",
		Example: `  sike op notify test 1`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			n, users, done, err := notifierService(cmd)
			if err != nil {
				return err
			}
			defer done()
			u, err := users.Get(cmd.Context(), id)
			if err != nil {
				return err
			}
			if err := n.Test(cmd.Context(), u, u.Topic); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "sent to %s\n", u.Topic)
			return nil
		},
	}

	var at string
	var dry bool
	tick := &cobra.Command{
		Use:   "tick [--at \"YYYY-MM-DD HH:MM\"] [--dry-run]",
		Short: "one tick of the notifier: send what is due in the 10 minutes up to now (or --at) and not sent yet",
		Example: `  sike op notify tick --dry-run
  sike op notify tick --dry-run --at "2026-10-14 18:00"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := location()
			if err != nil {
				return err
			}
			now := time.Now()
			if at != "" {
				if now, err = time.ParseInLocation("2006-01-02 15:04", at, loc); err != nil {
					return fmt.Errorf("--at %q: want YYYY-MM-DD HH:MM", at)
				}
			}
			if !dry && notifierPort() == nil {
				return fmt.Errorf("notifications are off: no SIKE_NTFY_URL (try --dry-run)")
			}
			n, _, done, err := notifierService(cmd)
			if err != nil {
				return err
			}
			defer done()
			notes, err := n.Tick(cmd.Context(), now, dry)
			for _, note := range notes {
				fmt.Fprintf(cmd.OutOrStdout(), "%s  %-12s  %s · %s\n", note.At.In(loc).Format("Mon 02 15:04"), note.Topic, note.Msg.Title, note.Msg.Body)
			}
			if err == nil && len(notes) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "nothing due")
			}
			return err
		},
	}
	tick.Flags().StringVar(&at, "at", "", "tick as if it were then (SIKE_TZ)")
	tick.Flags().BoolVar(&dry, "dry-run", false, "print what would be sent; send and record nothing")

	notify.AddCommand(test, tick)
	return notify
}

func notifierService(cmd *cobra.Command) (*service.Notifier, *service.Users, func(), error) {
	loc, err := location()
	if err != nil {
		return nil, nil, nil, err
	}
	st, done, err := openStore(cmd)
	if err != nil {
		return nil, nil, nil, err
	}
	users := service.NewUsers(st)
	return service.NewNotifier(notifierPort(), st, users, service.NewTasks(st), loc), users, done, nil
}

// notifierPort is ntfy at SIKE_NTFY_URL with SIKE_NTFY_TOKEN; none when the
// url is unset: notifications off.
func notifierPort() port.Notifier {
	url := getenv("SIKE_NTFY_URL", "")
	if url == "" {
		return nil
	}
	return ntfy.New(url, getenv("SIKE_NTFY_TOKEN", ""))
}
