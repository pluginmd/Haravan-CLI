package tools

import (
	"github.com/spf13/cobra"

	"github.com/pluginmd/haravan-cli/internal/cmdutil"
	"github.com/pluginmd/haravan-cli/internal/logger"
)

// categoryShort returns the short blurb shown for each top-level category
// command in --help output.
var categoryShort = map[Category]string{
	CatOrders:    "Order management (list, get, create, confirm, close, cancel, transactions)",
	CatProducts:  "Product and variant management",
	CatCustomers: "Customer and address management",
	CatInventory: "Inventory adjustments and location balances",
	CatShop:      "Shop info, locations, users, shipping rates",
	CatContent:   "Pages, blogs, articles, script tags",
	CatWebhooks:  "Webhook subscriptions",
	CatSmart:     "Server-side analytics and aggregations",
}

// MountAll adds one cobra subcommand per category to root, each of which
// contains the tools registered in the default Registry under that
// category. Call this once after all tool packages have been imported
// (their init() functions register themselves).
func MountAll(root *cobra.Command, f *cmdutil.Factory) {
	MountFromRegistry(root, f, Default())
}

// MountFromRegistry is the explicit-registry variant for tests.
func MountFromRegistry(root *cobra.Command, f *cmdutil.Factory, reg *Registry) {
	build := defaultDepsBuilder(f)

	for _, cat := range reg.Categories() {
		catCmd := &cobra.Command{
			Use:   string(cat),
			Short: categoryShort[cat],
		}
		for _, t := range reg.ByCategory(cat) {
			catCmd.AddCommand(BindCobra(t, f, build))
		}
		root.AddCommand(catCmd)
	}
}

func defaultDepsBuilder(f *cmdutil.Factory) DepsBuilder {
	return func(cmd *cobra.Command) (*Deps, error) {
		cfg, err := f.Config()
		if err != nil {
			return nil, err
		}
		c, err := cmdutil.BuildClient(cmd.Context(), cmd, cfg)
		if err != nil {
			return nil, err
		}
		return &Deps{
			Client: c,
			Logger: logger.Default(),
			IO:     f.IOStreams,
		}, nil
	}
}
