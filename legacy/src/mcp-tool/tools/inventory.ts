import { z } from 'zod';
import { McpTool } from '../types';

export const inventoryTools: McpTool[] = [
  {
    name: 'haravan_inventory_adjustments_list',
    project: 'inventory',
    description:
      'List inventory adjustments. Supports pagination and date filters.',
    schema: z.object({
      page: z.number().optional().describe('Page number'),
      limit: z.number().optional().describe('Results per page'),
      since_id: z.number().optional(),
      created_at_min: z.string().optional(),
      created_at_max: z.string().optional(),
      fetchAll: z.boolean().optional(),
    }),
    httpMethod: 'GET',
    path: '/com/inventories/adjustments.json',
    scopes: ['com.read_inventories'],
  },
  {
    name: 'haravan_inventory_adjustments_count',
    project: 'inventory',
    description: 'Count inventory adjustments.',
    schema: z.object({
      created_at_min: z.string().optional(),
      created_at_max: z.string().optional(),
    }),
    httpMethod: 'GET',
    path: '/com/inventories/adjustments/count.json',
    scopes: ['com.read_inventories'],
  },
  {
    name: 'haravan_inventory_adjustments_get',
    project: 'inventory',
    description: 'Get a single inventory adjustment by ID.',
    schema: z.object({
      adjustment_id: z.number().describe('Inventory adjustment ID'),
    }),
    httpMethod: 'GET',
    path: '/com/inventories/adjustments/{adjustment_id}.json',
    scopes: ['com.read_inventories'],
  },
  {
    name: 'haravan_inventory_adjust_or_set',
    project: 'inventory',
    description:
      'Create inventory adjustment. Type "adjust" adds/subtracts quantity. Type "set" replaces quantity. Max 200 line items per request.',
    schema: z.object({
      location_id: z.number().describe('Location/warehouse ID'),
      type: z
        .enum(['adjust', 'set'])
        .optional()
        .describe('adjust=add/subtract, set=replace (default: adjust)'),
      reason: z
        .enum([
          'newproduct',
          'returned',
          'productionofgoods',
          'damaged',
          'shrinkage',
          'promotion',
          'transfer',
        ])
        .optional()
        .describe('Adjustment reason'),
      note: z.string().optional().describe('Adjustment note'),
      tags: z.string().optional().describe('Comma-separated tags'),
      line_items: z
        .array(
          z.object({
            product_id: z.number().describe('Product ID'),
            product_variant_id: z.number().describe('Variant ID'),
            quantity: z.number().describe('Quantity to adjust/set'),
          })
        )
        .describe('Line items (max 200 per request)'),
    }),
    httpMethod: 'POST',
    path: '/com/inventories/adjustorset.json',
    scopes: ['com.write_inventories'],
  },
  {
    name: 'haravan_inventory_locations',
    project: 'inventory',
    description:
      'Get inventory levels by location and/or variant. Query stock balance across locations.',
    schema: z.object({
      location_id: z.number().optional().describe('Filter by location ID'),
      variant_id: z.number().optional().describe('Filter by variant ID'),
      product_id: z.number().optional().describe('Filter by product ID'),
      page: z.number().optional(),
      limit: z.number().optional(),
    }),
    httpMethod: 'GET',
    path: '/com/inventories/locations.json',
    scopes: ['com.read_inventories'],
  },
];
