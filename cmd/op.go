package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// op is the operator toolbox: everything the app does, from the terminal.
func opCmd() *cobra.Command {
	op := &cobra.Command{
		Use:   "op",
		Short: "operate on the data from the terminal",
	}
	op.AddCommand(opUserCmd(), opAreaCmd(), opTaskCmd(), opPlacementCmd(), opWeekCmd())
	return op
}

// confirm asks a yes/no question; only "y" or "yes" says yes.
func confirm(in io.Reader, out io.Writer, question string) bool {
	fmt.Fprintf(out, "%s [y/N] ", question)
	answer, _ := bufio.NewReader(in).ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}
