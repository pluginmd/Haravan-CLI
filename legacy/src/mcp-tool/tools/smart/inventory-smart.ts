import { z } from 'zod';
import { McpTool, MiddlewareContext } from '../../types';
import { clientFromCtx, fetchAll, ok, err, parseDate, sleep } from './helpers';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface Variant {
  id: number;
  title: string;
  sku?: string;
  price?: string;
  inventory_quantity?: number;
}

interface Product {
  id: number;
  title: string;
  product_type?: string;
  vendor?: string;
  variants: Variant[];
}

interface LineItem {
  variant_id: number;
  quantity: number;
}

interface Order {
  id: number;
  created_at: string;
  line_items: LineItem[];
}

interface InventoryLocation {
  variant_id: number;
  location_id: number;
  location_name?: string;
  qty_available: number;
  qty_onhand: number;
}

interface Location {
  id: number;
  name: string;
}

// ---------------------------------------------------------------------------
// Tool 1: hrv_inventory_health
// ---------------------------------------------------------------------------

const inventoryHealthHandler = async (ctx: MiddlewareContext) => {
  try {
    const { low_stock_threshold = 5, days_for_dead_stock = 90 } = ctx.params as {
      low_stock_threshold?: number;
      days_for_dead_stock?: number;
    };

    const client = clientFromCtx(ctx);

    // Step 1: fetch products (limit to first 100 for performance)
    const { items: allProducts, apiCalls: prodCalls } = await fetchAll<Product>(
      client,
      '/com/products.json',
      'products',
      {},
      { fields: 'id,title,variants,product_type,vendor', maxPages: 1 }
    );

    const products = allProducts.slice(0, 100);
    const hasMore = allProducts.length > 100;

    // Step 2: fetch inventory per variant (batch to respect rate limit)
    const variantInvMap = new Map<number, InventoryLocation[]>();
    let invApiCalls = 0;

    for (const product of products) {
      for (const variant of product.variants ?? []) {
        try {
          const resp = await client.get<any>('/com/inventories/locations.json', {
            variant_id: variant.id,
          });
          invApiCalls++;
          const locs: InventoryLocation[] = resp.data?.inventories ?? resp.data?.locations ?? [];
          variantInvMap.set(variant.id, locs);
          // Throttle every 20 calls
          if (invApiCalls % 20 === 0) await sleep(300);
        } catch {
          variantInvMap.set(variant.id, []);
        }
      }
    }

    // Step 3: fetch recent orders to detect sold variants
    const createdAtMin = parseDate(undefined, days_for_dead_stock);
    const { items: orders, apiCalls: orderCalls } = await fetchAll<Order>(
      client,
      '/com/orders.json',
      'orders',
      { created_at_min: createdAtMin, status: 'any' },
      { fields: 'id,line_items' }
    );

    const soldVariantIds = new Set<number>();
    for (const order of orders) {
      for (const li of order.line_items ?? []) {
        if (li.variant_id) soldVariantIds.add(li.variant_id);
      }
    }

    // Step 4: classify variants
    interface ClassifiedVariant {
      product_id: number;
      product_title: string;
      variant_id: number;
      variant_title: string;
      sku: string;
      qty_available: number;
      qty_onhand: number;
      price: number;
      category: 'out_of_stock' | 'low_stock' | 'dead_stock' | 'healthy';
    }

    const classified: ClassifiedVariant[] = [];

    for (const product of products) {
      for (const variant of product.variants ?? []) {
        const locs = variantInvMap.get(variant.id) ?? [];
        const qty_available = locs.reduce((s, l) => s + (l.qty_available ?? 0), 0);
        const qty_onhand = locs.reduce((s, l) => s + (l.qty_onhand ?? 0), 0);
        const price = parseFloat(variant.price ?? '0') || 0;

        let category: ClassifiedVariant['category'];
        if (qty_available === 0) {
          category = 'out_of_stock';
        } else if (qty_available < low_stock_threshold) {
          category = 'low_stock';
        } else if (qty_onhand > 0 && !soldVariantIds.has(variant.id)) {
          category = 'dead_stock';
        } else {
          category = 'healthy';
        }

        classified.push({
          product_id: product.id,
          product_title: product.title,
          variant_id: variant.id,
          variant_title: variant.title,
          sku: variant.sku ?? '',
          qty_available,
          qty_onhand,
          price,
          category,
        });
      }
    }

    const byCategory = (cat: ClassifiedVariant['category']) =>
      classified.filter((v) => v.category === cat);

    const deadStock = byCategory('dead_stock');
    const lowStock = byCategory('low_stock');

    const total_dead_stock_value = deadStock.reduce(
      (sum, v) => sum + v.qty_onhand * v.price,
      0
    );

    const top10LowStock = lowStock
      .sort((a, b) => a.qty_available - b.qty_available)
      .slice(0, 10)
      .map((v) => ({
        product_title: v.product_title,
        variant_title: v.variant_title,
        sku: v.sku,
        qty_available: v.qty_available,
      }));

    const top10DeadStock = deadStock
      .sort((a, b) => b.qty_onhand * b.price - a.qty_onhand * a.price)
      .slice(0, 10)
      .map((v) => ({
        product_title: v.product_title,
        variant_title: v.variant_title,
        sku: v.sku,
        qty_onhand: v.qty_onhand,
        dead_stock_value: +(v.qty_onhand * v.price).toFixed(2),
      }));

    return ok({
      summary: {
        total_variants_analyzed: classified.length,
        out_of_stock: byCategory('out_of_stock').length,
        low_stock: lowStock.length,
        healthy: byCategory('healthy').length,
        dead_stock: deadStock.length,
        total_dead_stock_value: +total_dead_stock_value.toFixed(2),
      },
      top_10_low_stock: top10LowStock,
      top_10_dead_stock: top10DeadStock,
      _meta: {
        products_analyzed: products.length,
        has_more_products: hasMore,
        orders_analyzed: orders.length,
        days_for_dead_stock,
        low_stock_threshold,
        api_calls: prodCalls + invApiCalls + orderCalls,
        note: hasMore
          ? 'Only first 100 products analyzed. Run with more specific filters for full coverage.'
          : undefined,
      },
    });
  } catch (e: any) {
    return err(`hrv_inventory_health failed: ${e?.message ?? e}`);
  }
};

// ---------------------------------------------------------------------------
// Tool 2: hrv_stock_reorder_plan
// ---------------------------------------------------------------------------

const stockReorderPlanHandler = async (ctx: MiddlewareContext) => {
  try {
    const {
      lead_time_days = 7,
      safety_factor = 1.3,
      date_range_days = 30,
    } = ctx.params as {
      lead_time_days?: number;
      safety_factor?: number;
      date_range_days?: number;
    };

    const client = clientFromCtx(ctx);

    // Step 1: fetch orders in range
    const createdAtMin = parseDate(undefined, date_range_days);
    const { items: orders } = await fetchAll<Order>(
      client,
      '/com/orders.json',
      'orders',
      { created_at_min: createdAtMin, status: 'any' },
      { fields: 'id,line_items,created_at' }
    );

    // Step 2: calculate sales per variant
    const salesMap = new Map<number, number>(); // variant_id -> total qty sold
    for (const order of orders) {
      for (const li of order.line_items ?? []) {
        if (li.variant_id) {
          salesMap.set(li.variant_id, (salesMap.get(li.variant_id) ?? 0) + (li.quantity ?? 1));
        }
      }
    }

    // Step 3: fetch products
    const { items: products } = await fetchAll<Product>(
      client,
      '/com/products.json',
      'products',
      {},
      { fields: 'id,title,variants' }
    );

    // Build variant lookup
    interface VariantInfo {
      product_id: number;
      product_title: string;
      variant_id: number;
      variant_title: string;
      sku: string;
    }
    const variantInfoMap = new Map<number, VariantInfo>();
    for (const product of products) {
      for (const variant of product.variants ?? []) {
        variantInfoMap.set(variant.id, {
          product_id: product.id,
          product_title: product.title,
          variant_id: variant.id,
          variant_title: variant.title,
          sku: variant.sku ?? '',
        });
      }
    }

    // Step 4: identify top-selling variant IDs (those with any sales)
    const sellingVariantIds = [...salesMap.entries()]
      .filter(([, qty]) => qty > 0)
      .sort((a, b) => b[1] - a[1])
      .map(([id]) => id);

    // Fetch inventory for top selling variants (up to 200 to keep calls sane)
    const variantInvMap = new Map<number, number>(); // variant_id -> qty_available
    let invCalls = 0;
    const toFetch = sellingVariantIds.slice(0, 200);

    for (const variantId of toFetch) {
      try {
        const resp = await client.get<any>('/com/inventories/locations.json', {
          variant_id: variantId,
        });
        invCalls++;
        const locs: InventoryLocation[] = resp.data?.inventories ?? resp.data?.locations ?? [];
        const qty_available = locs.reduce((s, l) => s + (l.qty_available ?? 0), 0);
        variantInvMap.set(variantId, qty_available);
        if (invCalls % 20 === 0) await sleep(300);
      } catch {
        variantInvMap.set(variantId, 0);
      }
    }

    // Step 5: calculate reorder metrics
    interface ReorderItem {
      product_title: string;
      variant_title: string;
      sku: string;
      qty_available: number;
      daily_sales_rate: number;
      days_of_stock: number | null;
      reorder_point: number;
      reorder_qty_suggested: number;
    }

    const reorderItems: ReorderItem[] = [];

    for (const variantId of toFetch) {
      const info = variantInfoMap.get(variantId);
      if (!info) continue;

      const total_sold = salesMap.get(variantId) ?? 0;
      const dsr = total_sold / date_range_days;
      const qty_available = variantInvMap.get(variantId) ?? 0;
      const reorder_point = dsr * lead_time_days * safety_factor;
      const reorder_qty = Math.max(0, Math.ceil(reorder_point - qty_available));
      const days_of_stock = dsr > 0 ? Math.floor(qty_available / dsr) : null;

      // Only include variants that actually need reorder or are critically low
      if (reorder_qty > 0 || (days_of_stock !== null && days_of_stock < lead_time_days * 2)) {
        reorderItems.push({
          product_title: info.product_title,
          variant_title: info.variant_title,
          sku: info.sku,
          qty_available,
          daily_sales_rate: +dsr.toFixed(3),
          days_of_stock,
          reorder_point: +reorder_point.toFixed(1),
          reorder_qty_suggested: reorder_qty,
        });
      }
    }

    // Sort by urgency: days_of_stock ASC (null treated as 0 = most urgent)
    reorderItems.sort((a, b) => {
      const da = a.days_of_stock ?? 0;
      const db = b.days_of_stock ?? 0;
      return da - db;
    });

    return ok({
      reorder_plan: reorderItems.slice(0, 30),
      _meta: {
        date_range_days,
        lead_time_days,
        safety_factor,
        orders_analyzed: orders.length,
        variants_with_sales: salesMap.size,
        variants_fetched_inventory: toFetch.length,
        total_needing_reorder: reorderItems.length,
        showing: Math.min(30, reorderItems.length),
      },
    });
  } catch (e: any) {
    return err(`hrv_stock_reorder_plan failed: ${e?.message ?? e}`);
  }
};

// ---------------------------------------------------------------------------
// Tool 3: hrv_inventory_imbalance
// ---------------------------------------------------------------------------

const inventoryImbalanceHandler = async (ctx: MiddlewareContext) => {
  try {
    const client = clientFromCtx(ctx);

    // Step 1: fetch locations
    const locResp = await client.get<any>('/com/locations.json', {});
    const locations: Location[] = locResp.data?.locations ?? [];

    if (locations.length < 2) {
      return ok({
        message: 'At least 2 locations are required to detect imbalance.',
        locations_found: locations.length,
        imbalanced_variants: [],
      });
    }

    // Step 2: fetch first 50 products
    const { items: allProducts } = await fetchAll<Product>(
      client,
      '/com/products.json',
      'products',
      {},
      { fields: 'id,title,variants', maxPages: 1 }
    );
    const products = allProducts.slice(0, 50);

    // Step 3: fetch inventory per variant (across all locations)
    interface VariantLocQty {
      location_id: number;
      location_name: string;
      qty_available: number;
    }

    interface VariantLocationData {
      product_title: string;
      variant_id: number;
      variant_title: string;
      sku: string;
      locations: VariantLocQty[];
    }

    const locationNameMap = new Map<number, string>();
    for (const loc of locations) locationNameMap.set(loc.id, loc.name);

    const variantLocationData: VariantLocationData[] = [];
    let invCalls = 0;

    for (const product of products) {
      for (const variant of product.variants ?? []) {
        try {
          const resp = await client.get<any>('/com/inventories/locations.json', {
            variant_id: variant.id,
          });
          invCalls++;
          const locs: InventoryLocation[] = resp.data?.inventories ?? resp.data?.locations ?? [];

          const locBreakdown: VariantLocQty[] = locs.map((l) => ({
            location_id: l.location_id,
            location_name: locationNameMap.get(l.location_id) ?? `Location ${l.location_id}`,
            qty_available: l.qty_available ?? 0,
          }));

          variantLocationData.push({
            product_title: product.title,
            variant_id: variant.id,
            variant_title: variant.title,
            sku: variant.sku ?? '',
            locations: locBreakdown,
          });

          if (invCalls % 20 === 0) await sleep(300);
        } catch {
          // skip variant on error
        }
      }
    }

    // Step 4: detect imbalance (max/min > 5 where min > 0)
    interface ImbalancedVariant {
      product_title: string;
      variant_title: string;
      sku: string;
      imbalance_ratio: number;
      locations: VariantLocQty[];
      suggested_transfer: {
        from_location: string;
        to_location: string;
        qty: number;
      };
    }

    const imbalanced: ImbalancedVariant[] = [];

    for (const vd of variantLocationData) {
      const withStock = vd.locations.filter((l) => l.qty_available > 0);
      if (withStock.length < 2) continue;

      const maxLoc = withStock.reduce((a, b) =>
        a.qty_available >= b.qty_available ? a : b
      );
      const minLoc = withStock.reduce((a, b) =>
        a.qty_available <= b.qty_available ? a : b
      );

      if (minLoc.qty_available === 0) continue;

      const ratio = maxLoc.qty_available / minLoc.qty_available;
      if (ratio <= 5) continue;

      // Suggest moving half the excess from max to min
      const target = Math.floor((maxLoc.qty_available + minLoc.qty_available) / 2);
      const transfer_qty = maxLoc.qty_available - target;

      imbalanced.push({
        product_title: vd.product_title,
        variant_title: vd.variant_title,
        sku: vd.sku,
        imbalance_ratio: +ratio.toFixed(1),
        locations: vd.locations.sort((a, b) => b.qty_available - a.qty_available),
        suggested_transfer: {
          from_location: maxLoc.location_name,
          to_location: minLoc.location_name,
          qty: transfer_qty,
        },
      });
    }

    // Sort by worst imbalance first
    imbalanced.sort((a, b) => b.imbalance_ratio - a.imbalance_ratio);

    return ok({
      imbalanced_variants: imbalanced,
      _meta: {
        locations_count: locations.length,
        products_analyzed: products.length,
        variants_analyzed: variantLocationData.length,
        imbalanced_count: imbalanced.length,
        imbalance_threshold: 5,
        api_calls: 1 + invCalls,
      },
    });
  } catch (e: any) {
    return err(`hrv_inventory_imbalance failed: ${e?.message ?? e}`);
  }
};

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

export const inventorySmartTools: McpTool[] = [
  {
    name: 'hrv_inventory_health',
    project: 'smart',
    description:
      'Analyze inventory health across products. Classifies variants as out_of_stock, low_stock, healthy, or dead_stock (items with stock but no sales in the given period). Returns summary counts, total dead-stock value, and top 10 lists. Analyzes first 100 products.',
    schema: z.object({
      low_stock_threshold: z
        .number()
        .optional()
        .default(5)
        .describe('Qty below which a variant is considered low stock (default 5)'),
      days_for_dead_stock: z
        .number()
        .optional()
        .default(90)
        .describe('Look-back window in days to determine if a variant sold (default 90)'),
    }),
    httpMethod: 'GET',
    path: '/smart',
    scopes: ['com.read_inventories', 'com.read_products'],
    customHandler: inventoryHealthHandler,
  },
  {
    name: 'hrv_stock_reorder_plan',
    project: 'smart',
    description:
      'Generate a stock reorder plan based on daily sales rate and current inventory. Calculates reorder point and suggested reorder quantity for each variant. Returns top 30 most urgent items sorted by days-of-stock remaining.',
    schema: z.object({
      lead_time_days: z
        .number()
        .optional()
        .default(7)
        .describe('Supplier lead time in days (default 7)'),
      safety_factor: z
        .number()
        .optional()
        .default(1.3)
        .describe('Safety buffer multiplier for reorder point (default 1.3)'),
      date_range_days: z
        .number()
        .optional()
        .default(30)
        .describe('Days of order history used to calculate daily sales rate (default 30)'),
    }),
    httpMethod: 'GET',
    path: '/smart',
    scopes: ['com.read_inventories', 'com.read_products'],
    customHandler: stockReorderPlanHandler,
  },
  {
    name: 'hrv_inventory_imbalance',
    project: 'smart',
    description:
      'Detect inventory imbalance across locations. Finds variants where the location with the most stock holds more than 5x the location with the least stock. Returns imbalanced variants with a suggested transfer to rebalance. Analyzes first 50 products.',
    schema: z.object({}),
    httpMethod: 'GET',
    path: '/smart',
    scopes: ['com.read_inventories', 'com.read_products'],
    customHandler: inventoryImbalanceHandler,
  },
];
