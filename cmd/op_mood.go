package cmd

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"sike/domain/clock"
	"sike/domain/model"
	"sike/domain/service"
)

func opMoodCmd() *cobra.Command {
	mood := &cobra.Command{
		Use:   "mood",
		Short: "moods: how a person's day went, 1 (😞) to 5 (😄)",
	}
	var date string
	// day is --date, or today (SIKE_TZ)
	day := func() (clock.Date, error) {
		if date != "" {
			return clock.ParseDate(date)
		}
		loc, err := location()
		if err != nil {
			return clock.Date{}, err
		}
		return clock.Today(time.Now(), loc), nil
	}
	// run opens the moods for the person in args[0], on the day
	run := func(f func(cmd *cobra.Command, args []string, moods *service.Moods, userID int64, d clock.Date) error) func(*cobra.Command, []string) error {
		return func(cmd *cobra.Command, args []string) error {
			userID, err := parseID(args[0])
			if err != nil {
				return err
			}
			d, err := day()
			if err != nil {
				return err
			}
			st, done, err := openStore(cmd)
			if err != nil {
				return err
			}
			defer done()
			return f(cmd, args, service.NewMoods(st), userID, d)
		}
	}

	show := &cobra.Command{
		Use:     "show <user-id>",
		Short:   "print a person's mood today (or on --date)",
		Example: `  sike op mood show 1`,
		Args:    cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string, moods *service.Moods, userID int64, d clock.Date) error {
			m, err := moods.Get(cmd.Context(), userID, d)
			if err != nil {
				return err
			}
			face := "none"
			if m > 0 {
				face = fmt.Sprintf("%d %s", m, model.Faces[m-1])
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", d, face)
			return nil
		}),
	}

	set := &cobra.Command{
		Use:   "set <user-id> <1-5 | face>",
		Short: "give a person's day its mood",
		Example: `  sike op mood set 1 4
  sike op mood set 1 🙂 --date 2026-10-14`,
		Args: cobra.ExactArgs(2),
		RunE: run(func(cmd *cobra.Command, args []string, moods *service.Moods, userID int64, d clock.Date) error {
			m, err := strconv.Atoi(args[1])
			if err != nil {
				m = slices.Index(model.Faces, args[1]) + 1 // a face, or 0: refused
			}
			return moods.Set(cmd.Context(), userID, d, m)
		}),
	}

	clear := &cobra.Command{
		Use:     "clear <user-id>",
		Short:   "take a day's mood away",
		Example: `  sike op mood clear 1`,
		Args:    cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string, moods *service.Moods, userID int64, d clock.Date) error {
			return moods.Clear(cmd.Context(), userID, d)
		}),
	}

	for _, c := range []*cobra.Command{show, set, clear} {
		c.Flags().StringVar(&date, "date", "", "the day, YYYY-MM-DD (default today)")
	}
	mood.AddCommand(show, set, clear)
	return mood
}
