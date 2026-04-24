package webhooks

import (
	"context"
	"errors"

	"github.com/pluginmd/haravan-cli/pkg/tools"
)

var whScopes = []string{"wh_api"}

// ErrNoWebhookClient is returned when the webhook client failed to initialize
// (usually because credentials couldn't be resolved).
var ErrNoWebhookClient = errors.New("webhook client not available; check credentials and HARAVAN_WEBHOOK_BASE")

func init() {
	tools.Register(&tools.Tool{
		Name:     "haravan_webhooks_list",
		Short:    "List subscribed webhooks",
		Long:     "List all webhook subscriptions for the current app (hits webhook.haravan.com, not the main API).",
		Category: tools.CatWebhooks,
		Scopes:   whScopes,
		Handler: func(ctx context.Context, deps *tools.Deps, _ tools.Input) (*tools.Result, error) {
			if deps.WebhookClient == nil {
				return nil, ErrNoWebhookClient
			}
			resp, err := deps.WebhookClient.Get(ctx, "/api/subscribe", nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_webhooks_subscribe",
		Short:    "Subscribe to a webhook topic",
		Long: `Subscribe the app to a webhook topic. Valid topics include:
orders/create, orders/updated, orders/paid, orders/cancelled, orders/fulfilled,
products/create, products/update, products/delete,
customers/create, customers/update, customers/delete,
shop/update, user/update, app/uninstalled.

The app must have a verified callback URL configured in the Developer Dashboard.`,
		Category: tools.CatWebhooks,
		Scopes:   whScopes,
		Flags: []tools.Flag{
			{Name: "topic", Type: tools.FlagString, Description: "Webhook topic, e.g. orders/create"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			if deps.WebhookClient == nil {
				return nil, ErrNoWebhookClient
			}
			var body any
			if topic := in.String("topic"); topic != "" {
				body = map[string]any{"topic": topic}
			}
			resp, err := deps.WebhookClient.Post(ctx, "/api/subscribe", body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_webhooks_unsubscribe",
		Short:    "Unsubscribe from a webhook topic",
		Category: tools.CatWebhooks,
		Scopes:   whScopes,
		Flags: []tools.Flag{
			{Name: "topic", Type: tools.FlagString, Description: "Webhook topic to unsubscribe"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			if deps.WebhookClient == nil {
				return nil, ErrNoWebhookClient
			}
			// Legacy parity: Haravan accepts the topic in the query string.
			q := tools.Query(in, "topic")
			resp, err := deps.WebhookClient.Request(ctx, "DELETE", "/api/subscribe", q, nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}
