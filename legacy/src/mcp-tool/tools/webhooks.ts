import { z } from 'zod';
import { McpTool } from '../types';

export const webhookTools: McpTool[] = [
  {
    name: 'haravan_webhooks_list',
    project: 'webhooks',
    description: 'List all subscribed webhooks for the current app.',
    schema: z.object({}),
    httpMethod: 'GET',
    path: '/api/subscribe',
    scopes: ['wh_api'],
    isWebhook: true,
  },
  {
    name: 'haravan_webhooks_subscribe',
    project: 'webhooks',
    description:
      'Subscribe to webhooks. The app must have a verified callback URL configured in the Developer Dashboard. Topics: orders/create, orders/updated, orders/paid, orders/cancelled, orders/fulfilled, products/create, products/update, products/delete, customers/create, customers/update, customers/delete, shop/update, user/update, app/uninstalled.',
    schema: z.object({
      topic: z
        .string()
        .optional()
        .describe(
          'Webhook topic to subscribe to (e.g., orders/create, products/update)'
        ),
    }),
    httpMethod: 'POST',
    path: '/api/subscribe',
    scopes: ['wh_api'],
    isWebhook: true,
  },
  {
    name: 'haravan_webhooks_unsubscribe',
    project: 'webhooks',
    description: 'Unsubscribe from webhooks.',
    schema: z.object({
      topic: z
        .string()
        .optional()
        .describe('Webhook topic to unsubscribe from'),
    }),
    httpMethod: 'DELETE',
    path: '/api/subscribe',
    scopes: ['wh_api'],
    isWebhook: true,
  },
];
