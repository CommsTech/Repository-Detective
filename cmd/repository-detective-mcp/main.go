// Command repository-detective-mcp is a stdio MCP bridge to the Repository Detective REST API.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"git.commsnet.org/commstech/repository-detective/mcpbridge"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg := mcpbridge.ConfigFromEnv()
	if err := mcpbridge.Serve(ctx, cfg, os.Stdin, os.Stdout); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "mcp bridge: %v\n", err)
		os.Exit(1)
	}
}
