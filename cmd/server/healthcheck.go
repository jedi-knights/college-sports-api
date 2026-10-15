package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// runHealthcheck implements the hidden `healthcheck` command used by
// docker-compose's HEALTHCHECK. It hits the local server's /readyz
// and exits nonzero when the readiness probe fails. Keeping it in-
// process avoids pulling curl/wget into the distroless image.
func runHealthcheck(cmd *cobra.Command, _ []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8080/readyz", http.NoBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: /readyz returned status %d", resp.StatusCode)
	}
	return nil
}
