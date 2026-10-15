package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"sike/domain/model"
	"sike/domain/service"
)

func opUserCmd() *cobra.Command {
	user := &cobra.Command{
		Use:   "user",
		Short: "people: each one is a separate world",
	}

	add := &cobra.Command{
		Use:     "add <name> <emoji>",
		Short:   "add a person",
		Example: `  sike op user add You 🦊`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			users, done, err := usersService(cmd)
			if err != nil {
				return err
			}
			defer done()
			u, err := users.Add(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			printUser(cmd, u)
			return nil
		},
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "list people, oldest first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			users, done, err := usersService(cmd)
			if err != nil {
				return err
			}
			defer done()
			all, err := users.List(cmd.Context())
			if err != nil {
				return err
			}
			for _, u := range all {
				printUser(cmd, u)
			}
			return nil
		},
	}

	var name, emoji, warn, topic string
	edit := &cobra.Command{
		Use:   "edit <id> [--name] [--emoji] [--warn 1h|1d|2d] [--topic]",
		Short: "change a person's name, emoji, deadline warning or ntfy topic; what is not given stays",
		Example: `  sike op user edit 1 --name Vic --emoji 🐸
  sike op user edit 1 --warn 2d --topic sike-vic`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			users, done, err := usersService(cmd)
			if err != nil {
				return err
			}
			defer done()
			u, err := users.Get(cmd.Context(), id)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				u.Name = name
			}
			if cmd.Flags().Changed("emoji") {
				u.Emoji = emoji
			}
			if cmd.Flags().Changed("topic") {
				u.Topic = topic
			}
			if cmd.Flags().Changed("warn") {
				w, ok := map[string]int{"1h": 60, "1d": 24 * 60, "2d": 48 * 60}[warn]
				if !ok {
					return fmt.Errorf("--warn is 1h, 1d or 2d")
				}
				u.Warn = w
			}
			if u, err = users.Edit(cmd.Context(), u); err != nil {
				return err
			}
			printUser(cmd, u)
			return nil
		},
	}
	edit.Flags().StringVar(&name, "name", "", "the new name")
	edit.Flags().StringVar(&emoji, "emoji", "", "the new emoji")
	edit.Flags().StringVar(&warn, "warn", "", "warn before deadlines: 1h, 1d or 2d")
	edit.Flags().StringVar(&topic, "topic", "", `their ntfy topic ("" for none)`)

	var yes bool
	rm := &cobra.Command{
		Use:   "rm <id>",
		Short: "delete a person and everything theirs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			users, done, err := usersService(cmd)
			if err != nil {
				return err
			}
			defer done()
			u, err := users.Get(cmd.Context(), id)
			if err != nil {
				return err
			}
			if !yes && !confirm(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("delete %s %s and everything theirs?", u.Emoji, u.Name)) {
				return fmt.Errorf("nothing deleted")
			}
			return users.Remove(cmd.Context(), id)
		},
	}
	rm.Flags().BoolVarP(&yes, "yes", "y", false, "don't ask")

	user.AddCommand(add, list, edit, rm)
	return user
}

// printUser: "1  🦊  You  warns 1 day before deadlines  ntfy sike-you"
func printUser(cmd *cobra.Command, u model.User) {
	topic := "no topic"
	if u.Topic != "" {
		topic = "ntfy " + u.Topic
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%d  %s  %s  warns %s before deadlines  %s\n", u.ID, u.Emoji, u.Name, model.WarnLabel(u.Warn), topic)
}

func usersService(cmd *cobra.Command) (*service.Users, func(), error) {
	st, done, err := openStore(cmd)
	if err != nil {
		return nil, nil, err
	}
	return service.NewUsers(st), done, nil
}
