import { McpTool } from './types';
import { customerTools, customerAddressTools } from './tools/customers';
import { orderTools, transactionTools } from './tools/orders';
import { productTools, variantTools } from './tools/products';
import { inventoryTools } from './tools/inventory';
import { shopTools } from './tools/shop';
import { webhookTools } from './tools/webhooks';
import { contentTools } from './tools/content';
// Smart tools (server-side aggregation — things Claude can't do efficiently)
import { orderSmartTools } from './tools/smart/orders-smart';
import { inventorySmartTools } from './tools/smart/inventory-smart';
import { customerSmartTools } from './tools/smart/customers-smart';

/**
 * All available Haravan MCP tools.
 *
 * Smart tools (hrv_*): server-side pagination + aggregation for large datasets.
 * Base tools (haravan_*): 1:1 API wrappers for detail/CRUD/drill-down.
 *
 * Simple aggregation (group by channel/province/discount) is handled by
 * Claude Skill layer, NOT by smart tools — avoids overlap.
 */
export const allTools: McpTool[] = [
  // Smart tools (7 total)
  ...orderSmartTools,
  ...inventorySmartTools,
  ...customerSmartTools,
  // Base API wrappers
  ...customerTools,
  ...customerAddressTools,
  ...orderTools,
  ...transactionTools,
  ...productTools,
  ...variantTools,
  ...inventoryTools,
  ...shopTools,
  ...webhookTools,
  ...contentTools,
];

/**
 * Get all unique project names.
 */
export function getProjects(): string[] {
  return [...new Set(allTools.map((t) => t.project))];
}

/**
 * Filter tools by project names, tool names, or preset names.
 */
export function filterTools(
  tools: McpTool[],
  filter?: string[]
): McpTool[] {
  if (!filter || filter.length === 0) {
    return tools;
  }

  const allowedNames = new Set<string>();
  const allowedProjects = new Set<string>();

  for (const entry of filter) {
    if (entry.startsWith('preset.')) {
      const presetTools = PRESETS[entry];
      if (presetTools) {
        presetTools.forEach((t) => allowedNames.add(t));
      }
      continue;
    }

    const matchingTools = tools.filter((t) => t.project === entry);
    if (matchingTools.length > 0) {
      allowedProjects.add(entry);
      continue;
    }

    allowedNames.add(entry);
  }

  if (filter.includes('all')) {
    return tools;
  }

  return tools.filter(
    (t) => allowedNames.has(t.name) || allowedProjects.has(t.project)
  );
}

/**
 * Tool presets for quick configuration.
 */
export const PRESETS: Record<string, string[]> = {
  // ===== Smart presets =====
  'preset.smart': [
    'hrv_orders_summary',
    'hrv_top_products',
    'hrv_order_cycle_time',
    'hrv_inventory_health',
    'hrv_stock_reorder_plan',
    'hrv_inventory_imbalance',
    'hrv_customer_segments',
  ],

  // ===== Default: Smart + detail tools for drill-down =====
  'preset.default': [
    // Smart aggregation
    'hrv_orders_summary',
    'hrv_top_products',
    'hrv_order_cycle_time',
    'hrv_inventory_health',
    'hrv_stock_reorder_plan',
    'hrv_inventory_imbalance',
    'hrv_customer_segments',
    // Detail tools for drill-down
    'haravan_shop_get',
    'haravan_locations_list',
    'haravan_products_list',
    'haravan_products_get',
    'haravan_orders_list',
    'haravan_orders_get',
    'haravan_customers_list',
    'haravan_customers_get',
    'haravan_customers_search',
    'haravan_inventory_locations',
  ],
  'preset.light': [
    'haravan_shop_get',
    'haravan_products_list',
    'haravan_products_get',
    'haravan_orders_list',
    'haravan_orders_get',
    'haravan_customers_list',
    'haravan_customers_get',
  ],

  // ===== Full CRUD presets =====
  'preset.products': [
    'haravan_products_list',
    'haravan_products_count',
    'haravan_products_get',
    'haravan_products_create',
    'haravan_products_update',
    'haravan_products_delete',
    'haravan_variants_list',
    'haravan_variants_count',
    'haravan_variants_get',
    'haravan_variants_create',
    'haravan_variants_update',
  ],
  'preset.orders': [
    'haravan_orders_list',
    'haravan_orders_count',
    'haravan_orders_get',
    'haravan_orders_create',
    'haravan_orders_update',
    'haravan_orders_confirm',
    'haravan_orders_close',
    'haravan_orders_open',
    'haravan_orders_cancel',
    'haravan_orders_assign',
    'haravan_transactions_list',
    'haravan_transactions_get',
    'haravan_transactions_create',
  ],
  'preset.customers': [
    'haravan_customers_list',
    'haravan_customers_search',
    'haravan_customers_count',
    'haravan_customers_get',
    'haravan_customers_create',
    'haravan_customers_update',
    'haravan_customers_delete',
    'haravan_customers_groups',
    'haravan_customer_addresses_list',
    'haravan_customer_addresses_get',
    'haravan_customer_addresses_create',
    'haravan_customer_addresses_update',
    'haravan_customer_addresses_delete',
    'haravan_customer_addresses_set_default',
  ],
  'preset.inventory': [
    'haravan_inventory_adjustments_list',
    'haravan_inventory_adjustments_count',
    'haravan_inventory_adjustments_get',
    'haravan_inventory_adjust_or_set',
    'haravan_inventory_locations',
    'haravan_locations_list',
    'haravan_locations_get',
  ],
  'preset.content': [
    'haravan_pages_list',
    'haravan_pages_get',
    'haravan_pages_create',
    'haravan_pages_update',
    'haravan_pages_delete',
    'haravan_blogs_list',
    'haravan_articles_list',
    'haravan_articles_get',
    'haravan_script_tags_list',
    'haravan_script_tags_create',
    'haravan_script_tags_delete',
  ],
  'preset.webhooks': [
    'haravan_webhooks_list',
    'haravan_webhooks_subscribe',
    'haravan_webhooks_unsubscribe',
  ],
};
