package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"sike/domain/model"
	"sike/domain/service"
)

func opTaskCmd() *cobra.Command {
	task := &cobra.Command{
		Use:   "task",
		Short: "tasks: done means deleted",
	}

	var area, when string
	var at []string
	add := &cobra.Command{
		Use:   "add <user-id> <name>",
		Short: "add a task, on top of its place in time",
		Example: `  sike op task add 1 "Fix bike" --area Home
  sike op task add 1 "Learn piano" --area lavender --when someday
  sike op task add 1 Gym --area Body --when weekly --at "mon,wed 18:00-19:30"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, err := parseID(args[0])
			if err != nil {
				return err
			}
			svc, done, err := taskServices(cmd)
			if err != nil {
				return err
			}
			defer done()
			color, err := svc.areaColor(cmd, userID, area)
			if err != nil {
				return err
			}
			places, err := parsePlaces(at)
			if err != nil {
				return err
			}
			t, err := svc.tasks.Add(cmd.Context(), userID, args[1], color, model.When(when), places)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d  %s\n", t.ID, t.Name)
			return nil
		},
	}
	add.Flags().StringVar(&area, "area", "", "the area: its color or its name (required)")
	add.Flags().StringVar(&when, "when", string(model.Pending), "weekly, pending or someday")
	add.Flags().StringArrayVar(&at, "at", nil, `a weekly time, "mon,wed 18:00-19:30" (repeat it for more)`)
	_ = add.MarkFlagRequired("area")

	list := &cobra.Command{
		Use:   "list <user-id>",
		Short: "list a person's tasks, as the tabs order them",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, err := parseID(args[0])
			if err != nil {
				return err
			}
			svc, done, err := taskServices(cmd)
			if err != nil {
				return err
			}
			defer done()
			areas, err := svc.areas.List(cmd.Context(), userID)
			if err != nil {
				return err
			}
			names := map[string]string{}
			for _, a := range areas {
				names[a.Color] = a.Name
			}
			tasks, err := svc.tasks.List(cmd.Context(), userID)
			if err != nil {
				return err
			}
			for _, t := range tasks {
				fmt.Fprintf(cmd.OutOrStdout(), "%d  %-8s  %-8s  %s\n", t.ID, t.When, names[t.Color], t.Name)
				if t.When != model.Weekly {
					continue
				}
				places, err := svc.tasks.Placements(cmd.Context(), userID, t.ID)
				if err != nil {
					return err
				}
				for _, p := range places {
					fmt.Fprintf(cmd.OutOrStdout(), "     %s\n", formatPlace(p))
				}
			}
			return nil
		},
	}

	rm := &cobra.Command{
		Use:   "rm <user-id> <task-id>",
		Short: "delete a task: it is done (restore undoes the last one)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, id, err := parseIDs(args[0], args[1])
			if err != nil {
				return err
			}
			svc, done, err := taskServices(cmd)
			if err != nil {
				return err
			}
			defer done()
			return svc.tasks.Remove(cmd.Context(), userID, id)
		},
	}

	restore := &cobra.Command{
		Use:   "restore <user-id> <task-id>",
		Short: "undo a delete: only a person's last one can be undone",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, id, err := parseIDs(args[0], args[1])
			if err != nil {
				return err
			}
			svc, done, err := taskServices(cmd)
			if err != nil {
				return err
			}
			defer done()
			return svc.tasks.Restore(cmd.Context(), userID, id)
		},
	}

	move := &cobra.Command{
		Use:     "move <user-id> <task-id> <when>",
		Short:   "put a task in pending or someday, on top (weekly needs times: edit)",
		Example: `  sike op task move 1 4 someday`,
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, id, err := parseIDs(args[0], args[1])
			if err != nil {
				return err
			}
			svc, done, err := taskServices(cmd)
			if err != nil {
				return err
			}
			defer done()
			return svc.tasks.Move(cmd.Context(), userID, id, model.When(args[2]))
		},
	}

	var name, editArea, editWhen string
	var editAt []string
	edit := &cobra.Command{
		Use:   "edit <user-id> <task-id>",
		Short: "change a task's name, area, when or weekly times; a new when puts it on top there",
		Example: `  sike op task edit 1 4 --name "Fix the bike"
  sike op task edit 1 4 --area Fun --when someday
  sike op task edit 1 4 --when weekly --at "sat 10:00-12:00"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, id, err := parseIDs(args[0], args[1])
			if err != nil {
				return err
			}
			svc, done, err := taskServices(cmd)
			if err != nil {
				return err
			}
			defer done()
			t, err := svc.tasks.Get(cmd.Context(), userID, id)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				t.Name = name
			}
			if cmd.Flags().Changed("area") {
				if t.Color, err = svc.areaColor(cmd, userID, editArea); err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("when") {
				t.When = model.When(editWhen)
			}
			places, err := svc.tasks.Placements(cmd.Context(), userID, id)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("at") {
				if places, err = parsePlaces(editAt); err != nil {
					return err
				}
			}
			t, err = svc.tasks.Edit(cmd.Context(), userID, id, t.Name, t.Color, t.When, places)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d  %s  %s  %s\n", t.ID, t.When, t.Color, t.Name)
			return nil
		},
	}
	edit.Flags().StringVar(&name, "name", "", "the new name")
	edit.Flags().StringVar(&editArea, "area", "", "the new area: its color or its name")
	edit.Flags().StringVar(&editWhen, "when", "", "weekly, pending or someday")
	edit.Flags().StringArrayVar(&editAt, "at", nil, `replaces the weekly times: "mon,wed 18:00-19:30" (repeat it for more)`)
	edit.MarkFlagsOneRequired("name", "area", "when", "at")

	task.AddCommand(add, list, edit, rm, restore, move)
	return task
}

type taskSvc struct {
	tasks *service.Tasks
	areas *service.Areas
}

func taskServices(cmd *cobra.Command) (taskSvc, func(), error) {
	st, done, err := openStore(cmd)
	if err != nil {
		return taskSvc{}, nil, err
	}
	return taskSvc{tasks: service.NewTasks(st), areas: service.NewAreas(st)}, done, nil
}

// areaColor finds a person's area by its color or its name, ignoring case.
func (s taskSvc) areaColor(cmd *cobra.Command, userID int64, area string) (string, error) {
	areas, err := s.areas.List(cmd.Context(), userID)
	if err != nil {
		return "", err
	}
	var names []string
	for _, a := range areas {
		if strings.EqualFold(area, a.Color) || strings.EqualFold(area, a.Name) {
			return a.Color, nil
		}
		names = append(names, a.Name)
	}
	return "", fmt.Errorf("no area %q: pick one of %s, or a color", area, strings.Join(names, ", "))
}

func parsePlaces(specs []string) ([]model.Placement, error) {
	var places []model.Placement
	for _, spec := range specs {
		p, err := parsePlace(spec)
		if err != nil {
			return nil, err
		}
		places = append(places, p)
	}
	return places, nil
}

func parseIDs(a, b string) (int64, int64, error) {
	x, err := parseID(a)
	if err != nil {
		return 0, 0, err
	}
	y, err := parseID(b)
	return x, y, err
}
