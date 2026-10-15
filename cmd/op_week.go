package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"sike/domain/clock"
)

func opWeekCmd() *cobra.Command {
	week := &cobra.Command{
		Use:   "week",
		Short: "the week, as the time core sees it",
	}

	var date string
	show := &cobra.Command{
		Use:   "show",
		Short: "print a week: its days and the timeline's hours (SIKE_TZ)",
		Example: `  sike op week show
  sike op week show --date 2026-03-29`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			loc, err := location()
			if err != nil {
				return err
			}
			today := clock.Today(time.Now(), loc)
			day := today
			if date != "" {
				if day, err = clock.ParseDate(date); err != nil {
					return err
				}
			}
			monday := day.Monday()
			sunday := monday.AddDays(6)
			// placements come with m15: nothing to expand yet
			occs := clock.Expand(nil, monday, sunday.AddDays(1))
			from, to := clock.HourRange(occs)

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "week %d · %s – %s · %s\n", monday.ISOWeek(), monday, sunday, loc)
			fmt.Fprintf(out, "hours %02d–%02d\n", from, to)
			for d := monday; d.Before(sunday.AddDays(1)); d = d.AddDays(1) {
				mark := ""
				if d == today {
					mark = "  today"
				}
				fmt.Fprintf(out, "%s %s%s\n", d.Weekday().String()[:3], d, mark)
				for _, o := range occs {
					if o.Date == d {
						fmt.Fprintf(out, "    %s–%s  %d\n", o.Start, o.End, o.Ref)
					}
				}
			}
			return nil
		},
	}
	show.Flags().StringVar(&date, "date", "", "any day of the week to show (YYYY-MM-DD); default today")

	week.AddCommand(show)
	return week
}

// location is SIKE_TZ (default Europe/Madrid): the one place all times are
// wall-clock in.
func location() (*time.Location, error) {
	name := getenv("SIKE_TZ", "Europe/Madrid")
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("SIKE_TZ %q: %w", name, err)
	}
	return loc, nil
}
