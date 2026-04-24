package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	mcpsrv "github.com/mark3labs/mcp-go/server"
)

// TransportConfig picks between stdio (default) and HTTP/SSE.
type TransportConfig struct {
	HTTP    bool   // if true, run the HTTP/SSE transport; otherwise stdio
	Addr    string // e.g. ":4567" — used only when HTTP is true
	BaseURL string // public URL for SSE endpoints (optional)
}

// Serve runs the MCP server over the selected transport, returning when the
// transport stops (stdio) or the ctx is cancelled (HTTP).
func Serve(ctx context.Context, srv *mcpsrv.MCPServer, cfg TransportConfig) error {
	if !cfg.HTTP {
		return mcpsrv.ServeStdio(srv)
	}
	addr := cfg.Addr
	if addr == "" {
		addr = ":4567"
	}
	opts := []mcpsrv.SSEOption{}
	if cfg.BaseURL != "" {
		opts = append(opts, mcpsrv.WithBaseURL(cfg.BaseURL))
	}
	sse := mcpsrv.NewSSEServer(srv, opts...)

	errCh := make(chan error, 1)
	go func() { errCh <- sse.Start(addr) }()

	select {
	case <-ctx.Done():
		// Give the server a moment to wind down.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := sse.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("sse shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		if err == nil || err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("sse server: %w", err)
	}
}
