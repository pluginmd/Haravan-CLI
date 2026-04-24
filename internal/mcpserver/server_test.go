package mcpserver_test

import (
	"testing"

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
	// Just verify NewServer succeeds — it iterates every tool, which in turn
	// builds a schema for each. If any tool has a broken flag config, this
	// catches it at test time.
	if _, err := mcpserver.NewServer(mcpserver.Options{Client: c}); err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if n := tools.Default().Count(); n != 70 {
		t.Errorf("tool count: got %d, want 70", n)
	}
}
