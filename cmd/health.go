package cmd

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// healthCmd asks a running server whether it is well: the image's healthcheck
// (distroless has no curl).
func healthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "check that the server on SIKE_ADDR answers and its db is there (exit 1 if not)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := getenv("SIKE_ADDR", ":8090")
			if strings.HasPrefix(addr, ":") {
				addr = "127.0.0.1" + addr
			}
			c := http.Client{Timeout: 3 * time.Second}
			res, err := c.Get("http://" + addr + "/healthz")
			if err != nil {
				return err
			}
			defer res.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("%s: %s", res.Status, strings.TrimSpace(string(body)))
			}
			fmt.Fprint(cmd.OutOrStdout(), string(body))
			return nil
		},
	}
}
