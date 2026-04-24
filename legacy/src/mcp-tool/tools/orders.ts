import { z } from 'zod';
import { McpTool } from '../types';

const lineItemSchema = z.object({
  variant_id: z.number().optional().describe('Product variant ID'),
  product_id: z.number().optional().describe('Product ID'),
  title: z.string().optional().describe('Product title'),
  quantity: z.number().describe('Quantity'),
  price: z.number().optional().describe('Unit price'),
  sku: z.string().optional().describe('SKU'),
  grams: z.number().optional().describe('Weight in grams'),
  requires_shipping: z.boolean().optional(),
  taxable: z.boolean().optional(),
});

export const orderTools: McpTool[] = [
  {
    name: 'haravan_orders_list',
    project: 'orders',
    description:
      'List orders. Filter by status, financial_status, fulfillment_status, created_at, updated_at. Supports pagination.',
    schema: z.object({
      page: z.number().optional().describe('Page number (default: 1)'),
      limit: z.number().optional().describe('Results per page (default: 50, max: 250)'),
      since_id: z.number().optional().describe('Results after this ID'),
      created_at_min: z.string().optional().describe('Created after (ISO 8601)'),
      created_at_max: z.string().optional().describe('Created before (ISO 8601)'),
      updated_at_min: z.string().optional().describe('Updated after (ISO 8601)'),
      updated_at_max: z.string().optional().describe('Updated before (ISO 8601)'),
      status: z
        .enum(['open', 'closed', 'cancelled', 'any'])
        .optional()
        .describe('Order status filter'),
      financial_status: z
        .enum([
          'pending',
          'authorized',
          'partially_paid',
          'paid',
          'partially_refunded',
          'refunded',
          'voided',
          'any',
        ])
        .optional()
        .describe('Financial status filter'),
      fulfillment_status: z
        .enum(['fulfilled', 'partial', 'unshipped', 'any'])
        .optional()
        .describe('Fulfillment status filter'),
      fields: z.string().optional().describe('Comma-separated fields'),
      fetchAll: z.boolean().optional().describe('Auto-paginate all results'),
    }),
    httpMethod: 'GET',
    path: '/com/orders.json',
    scopes: ['com.read_orders'],
  },
  {
    name: 'haravan_orders_count',
    project: 'orders',
    description: 'Get total order count with optional filters.',
    schema: z.object({
      status: z.enum(['open', 'closed', 'cancelled', 'any']).optional(),
      financial_status: z.string().optional(),
      fulfillment_status: z.string().optional(),
      created_at_min: z.string().optional(),
      created_at_max: z.string().optional(),
      updated_at_min: z.string().optional(),
      updated_at_max: z.string().optional(),
    }),
    httpMethod: 'GET',
    path: '/com/orders/count.json',
    scopes: ['com.read_orders'],
  },
  {
    name: 'haravan_orders_get',
    project: 'orders',
    description:
      'Get a single order by ID. Returns full order details including line_items, shipping, billing, transactions.',
    schema: z.object({
      order_id: z.number().describe('Order ID'),
      fields: z.string().optional().describe('Comma-separated fields'),
    }),
    httpMethod: 'GET',
    path: '/com/orders/{order_id}.json',
    scopes: ['com.read_orders'],
  },
  {
    name: 'haravan_orders_create',
    project: 'orders',
    description:
      'Create a new order. Requires line_items. Supports billing/shipping addresses, discounts, notes, tags.',
    schema: z.object({
      order: z
        .object({
          line_items: z.array(lineItemSchema).describe('Order line items'),
          email: z.string().optional().describe('Customer email'),
          phone: z.string().optional().describe('Customer phone'),
          billing_address: z
            .object({
              first_name: z.string().optional(),
              last_name: z.string().optional(),
              address1: z.string().optional(),
              city: z.string().optional(),
              province: z.string().optional(),
              country: z.string().optional(),
              phone: z.string().optional(),
              zip: z.string().optional(),
            })
            .optional(),
          shipping_address: z
            .object({
              first_name: z.string().optional(),
              last_name: z.string().optional(),
              address1: z.string().optional(),
              city: z.string().optional(),
              province: z.string().optional(),
              country: z.string().optional(),
              phone: z.string().optional(),
              zip: z.string().optional(),
            })
            .optional(),
          note: z.string().optional().describe('Order note'),
          tags: z.string().optional().describe('Comma-separated tags'),
          discount_codes: z
            .array(
              z.object({
                code: z.string(),
                amount: z.number(),
                type: z.string().optional(),
              })
            )
            .optional(),
          financial_status: z.string().optional(),
          source_name: z.string().optional().describe('Order source: web, pos, etc.'),
          send_receipt: z.boolean().optional().describe('Send receipt email'),
          note_attributes: z
            .array(z.object({ name: z.string(), value: z.string() }))
            .optional(),
        })
        .describe('Order object'),
    }),
    httpMethod: 'POST',
    path: '/com/orders.json',
    scopes: ['com.write_orders'],
  },
  {
    name: 'haravan_orders_update',
    project: 'orders',
    description: 'Update an existing order. Can update note, tags, shipping_address, email, etc.',
    schema: z.object({
      order_id: z.number().describe('Order ID'),
      order: z
        .object({
          id: z.number().describe('Order ID'),
          note: z.string().optional(),
          tags: z.string().optional(),
          email: z.string().optional(),
          phone: z.string().optional(),
          shipping_address: z
            .object({
              first_name: z.string().optional(),
              last_name: z.string().optional(),
              address1: z.string().optional(),
              city: z.string().optional(),
              province: z.string().optional(),
              country: z.string().optional(),
              phone: z.string().optional(),
            })
            .optional(),
        })
        .describe('Order fields to update'),
    }),
    httpMethod: 'PUT',
    path: '/com/orders/{order_id}.json',
    scopes: ['com.write_orders'],
  },
  {
    name: 'haravan_orders_confirm',
    project: 'orders',
    description: 'Confirm an order.',
    schema: z.object({
      order_id: z.number().describe('Order ID to confirm'),
    }),
    httpMethod: 'POST',
    path: '/com/orders/{order_id}/confirm.json',
    scopes: ['com.write_orders'],
  },
  {
    name: 'haravan_orders_close',
    project: 'orders',
    description: 'Close an order.',
    schema: z.object({
      order_id: z.number().describe('Order ID to close'),
    }),
    httpMethod: 'POST',
    path: '/com/orders/{order_id}/close.json',
    scopes: ['com.write_orders'],
  },
  {
    name: 'haravan_orders_open',
    project: 'orders',
    description: 'Reopen a closed order.',
    schema: z.object({
      order_id: z.number().describe('Order ID to reopen'),
    }),
    httpMethod: 'POST',
    path: '/com/orders/{order_id}/open.json',
    scopes: ['com.write_orders'],
  },
  {
    name: 'haravan_orders_cancel',
    project: 'orders',
    description:
      'Cancel an order. Provide reason: customer, fraud, inventory, declined, other.',
    schema: z.object({
      order_id: z.number().describe('Order ID to cancel'),
      reason: z
        .enum(['customer', 'fraud', 'inventory', 'declined', 'other'])
        .optional()
        .describe('Cancellation reason'),
      email: z.boolean().optional().describe('Send cancellation email'),
      restock: z.boolean().optional().describe('Restock cancelled items'),
    }),
    httpMethod: 'POST',
    path: '/com/orders/{order_id}/cancel.json',
    scopes: ['com.write_orders'],
  },
  {
    name: 'haravan_orders_assign',
    project: 'orders',
    description: 'Assign staff to an order.',
    schema: z.object({
      order_id: z.number().describe('Order ID'),
      user_id: z.number().describe('Staff user ID to assign'),
    }),
    httpMethod: 'POST',
    path: '/com/orders/{order_id}/assign.json',
    scopes: ['com.write_orders'],
  },
];

export const transactionTools: McpTool[] = [
  {
    name: 'haravan_transactions_list',
    project: 'orders',
    description: 'List all transactions for an order.',
    schema: z.object({
      order_id: z.number().describe('Order ID'),
    }),
    httpMethod: 'GET',
    path: '/com/orders/{order_id}/transactions.json',
    scopes: ['com.read_orders'],
  },
  {
    name: 'haravan_transactions_get',
    project: 'orders',
    description: 'Get a specific transaction.',
    schema: z.object({
      order_id: z.number().describe('Order ID'),
      transaction_id: z.number().describe('Transaction ID'),
    }),
    httpMethod: 'GET',
    path: '/com/orders/{order_id}/transactions/{transaction_id}.json',
    scopes: ['com.read_orders'],
  },
  {
    name: 'haravan_transactions_create',
    project: 'orders',
    description:
      'Create a transaction for an order. Kind: Pending, Authorization, Sale, Capture, Void, Refund.',
    schema: z.object({
      order_id: z.number().describe('Order ID'),
      transaction: z
        .object({
          amount: z.number().describe('Transaction amount'),
          kind: z
            .enum(['Pending', 'Authorization', 'Sale', 'Capture', 'Void', 'Refund'])
            .describe('Transaction kind'),
          gateway: z.string().optional().describe('Payment gateway'),
          currency: z.string().optional().describe('Currency code (ISO 4217)'),
        })
        .describe('Transaction object'),
    }),
    httpMethod: 'POST',
    path: '/com/orders/{order_id}/transactions.json',
    scopes: ['com.write_orders'],
  },
];
