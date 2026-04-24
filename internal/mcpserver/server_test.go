package mcpserver_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/pluginmd/haravan-cli/internal/client"
	"github.com/pluginmd/haravan-cli/internal/mcpserver"
	"github.com/pluginmd/haravan-cli/pkg/tools"

	// Populate tools.Default() at test time.
	_ "github.com/pluginmd/haravan-cli/pkg/tools/content"
	_ "github.com/pluginmd/haravan-cli/pkg/tools/customers"
	_ "github.com/pluginmd/haravan-cli/pkg/tools/inventory"
	_ "github.com/pluginmd/haravan-cli/pkg/tools/orders"
	_ "github.com/pluginmd/haravan-cli/pkg/tools/products"
	_ "github.com/pluginmd/haravan-cli/pkg/tools/shop"
	_ "github.com/pluginmd/haravan-cli/pkg/tools/smart"
	_ "github.com/pluginmd/haravan-cli/pkg/tools/webhooks"
)

func TestNewServerRegistersEveryRegisteredTool(t *testing.T) {
	c, err := client.New(client.Options{BaseURL: "https://example.invalid", AccessToken: "t"})
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	if _, err := mcpserver.NewServer(mcpserver.Options{Client: c}); err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if n := tools.Default().Count(); n != 70 {
		t.Errorf("tool count: got %d, want 70", n)
	}
}

// TestMCPHandshakeAndListTools simulates a Claude Desktop-style client:
// initialize → initialized → tools/list and verifies the server returns
// a well-formed response containing every registered tool with a valid
// JSON-Schema input_schema.
//
// This catches the RawInputSchema vs InputSchema marshal conflict that
// otherwise only surfaces on a live tools/list call.
func TestMCPHandshakeAndListTools(t *testing.T) {
	c, err := client.New(client.Options{BaseURL: "https://example.invalid", AccessToken: "t"})
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	srv, err := mcpserver.NewServer(mcpserver.Options{Client: c})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initReq := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`)
	initResp := srv.HandleMessage(ctx, initReq)
	initBytes, err := json.Marshal(initResp)
	if err != nil {
		t.Fatalf("marshal initialize response: %v", err)
	}
	var initEnv struct {
		Result struct {
			ProtocolVersion string         `json:"protocolVersion"`
			ServerInfo      map[string]any `json:"serverInfo"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(initBytes, &initEnv); err != nil {
		t.Fatalf("decode initialize: %v; body=%s", err, initBytes)
	}
	if initEnv.Error != nil {
		t.Fatalf("initialize error: %s", initEnv.Error.Message)
	}
	if initEnv.Result.ProtocolVersion == "" {
		t.Errorf("initialize missing protocolVersion: %s", initBytes)
	}
	if initEnv.Result.ServerInfo == nil {
		t.Errorf("initialize missing serverInfo: %s", initBytes)
	}

	listResp := srv.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	listBytes, err := json.Marshal(listResp)
	if err != nil {
		t.Fatalf("marshal tools/list response: %v", err)
	}
	var env struct {
		Result struct {
			Tools []mcp.Tool `json:"tools"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(listBytes, &env); err != nil {
		t.Fatalf("decode tools/list: %v; body=%s", err, listBytes)
	}
	if env.Error != nil {
		t.Fatalf("tools/list error: %s", env.Error.Message)
	}
	if n := len(env.Result.Tools); n != 70 {
		t.Errorf("tools count: got %d, want 70", n)
	}
	// Every tool must marshal cleanly — this is what would fail if InputSchema
	// and RawInputSchema conflicted at the marshal layer.
	for _, tt := range env.Result.Tools {
		if _, err := json.Marshal(tt); err != nil {
			t.Errorf("tool %q failed to re-marshal: %v", tt.Name, err)
		}
	}
}
