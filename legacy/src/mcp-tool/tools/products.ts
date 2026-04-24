import { z } from 'zod';
import { McpTool } from '../types';

export const productTools: McpTool[] = [
  {
    name: 'haravan_products_list',
    project: 'products',
    description:
      'List all products. Supports pagination, filtering by collection_id, product_type, vendor, handle, created_at, updated_at, published_at, published_status.',
    schema: z.object({
      page: z.number().optional().describe('Page number'),
      limit: z.number().optional().describe('Results per page (max: 250)'),
      since_id: z.number().optional().describe('Results after this ID'),
      collection_id: z.number().optional().describe('Filter by collection'),
      product_type: z.string().optional().describe('Filter by product type'),
      vendor: z.string().optional().describe('Filter by vendor'),
      handle: z.string().optional().describe('Filter by handle (URL slug)'),
      created_at_min: z.string().optional(),
      created_at_max: z.string().optional(),
      updated_at_min: z.string().optional(),
      updated_at_max: z.string().optional(),
      published_at_min: z.string().optional(),
      published_at_max: z.string().optional(),
      published_status: z
        .enum(['published', 'unpublished', 'any'])
        .optional()
        .describe('Published status'),
      fields: z.string().optional().describe('Comma-separated fields'),
      fetchAll: z.boolean().optional().describe('Auto-paginate all results'),
    }),
    httpMethod: 'GET',
    path: '/com/products.json',
    scopes: ['com.read_products'],
  },
  {
    name: 'haravan_products_count',
    project: 'products',
    description: 'Get total product count with optional filters.',
    schema: z.object({
      collection_id: z.number().optional(),
      product_type: z.string().optional(),
      vendor: z.string().optional(),
      created_at_min: z.string().optional(),
      created_at_max: z.string().optional(),
      updated_at_min: z.string().optional(),
      updated_at_max: z.string().optional(),
      published_at_min: z.string().optional(),
      published_at_max: z.string().optional(),
      published_status: z.enum(['published', 'unpublished', 'any']).optional(),
    }),
    httpMethod: 'GET',
    path: '/com/products/count.json',
    scopes: ['com.read_products'],
  },
  {
    name: 'haravan_products_get',
    project: 'products',
    description:
      'Get a single product by ID. Returns full details including variants, images, options.',
    schema: z.object({
      product_id: z.number().describe('Product ID'),
      fields: z.string().optional().describe('Comma-separated fields'),
    }),
    httpMethod: 'GET',
    path: '/com/products/{product_id}.json',
    scopes: ['com.read_products'],
  },
  {
    name: 'haravan_products_create',
    project: 'products',
    description:
      'Create a new product. Set title, body_html, vendor, product_type, tags, variants, images, options.',
    schema: z.object({
      product: z
        .object({
          title: z.string().describe('Product title'),
          body_html: z.string().optional().describe('HTML description'),
          vendor: z.string().optional().describe('Vendor/supplier name'),
          product_type: z.string().optional().describe('Product type for filtering'),
          tags: z.string().optional().describe('Comma-separated tags (max 250)'),
          published: z.boolean().optional().describe('Published status'),
          published_scope: z
            .enum(['web', 'pos', 'global'])
            .optional()
            .describe('Where product is visible'),
          template_suffix: z.string().optional(),
          only_hide_from_list: z
            .boolean()
            .optional()
            .describe('Hide from search/listing'),
          not_allow_promotion: z
            .boolean()
            .optional()
            .describe('Disallow promotions/coupons'),
          variants: z
            .array(
              z.object({
                title: z.string().optional(),
                price: z.number().optional(),
                compare_at_price: z.number().optional(),
                sku: z.string().optional(),
                barcode: z.string().optional(),
                grams: z.number().optional(),
                inventory_management: z
                  .enum(['haravan'])
                  .nullable()
                  .optional()
                  .describe('null or "haravan"'),
                inventory_policy: z
                  .enum(['deny', 'continue'])
                  .optional()
                  .describe('deny=stop selling when 0, continue=allow oversell'),
                inventory_quantity: z.number().optional(),
                requires_shipping: z.boolean().optional(),
                taxable: z.boolean().optional(),
                option1: z.string().optional(),
                option2: z.string().optional(),
                option3: z.string().optional(),
              })
            )
            .optional()
            .describe('Product variants (max 100)'),
          images: z
            .array(
              z.object({
                src: z.string().describe('Image URL'),
                position: z.number().optional(),
                variant_ids: z.array(z.number()).optional(),
              })
            )
            .optional()
            .describe('Product images'),
          options: z
            .array(
              z.object({
                name: z.string().describe('Option name (e.g., Size, Color)'),
                values: z.array(z.string()).optional(),
              })
            )
            .optional()
            .describe('Product options (max 3)'),
        })
        .describe('Product object'),
    }),
    httpMethod: 'POST',
    path: '/com/products.json',
    scopes: ['com.write_products'],
  },
  {
    name: 'haravan_products_update',
    project: 'products',
    description: 'Update an existing product.',
    schema: z.object({
      product_id: z.number().describe('Product ID'),
      product: z
        .object({
          id: z.number().describe('Product ID'),
          title: z.string().optional(),
          body_html: z.string().optional(),
          vendor: z.string().optional(),
          product_type: z.string().optional(),
          tags: z.string().optional(),
          published: z.boolean().optional(),
          template_suffix: z.string().optional(),
          only_hide_from_list: z.boolean().optional(),
          not_allow_promotion: z.boolean().optional(),
          variants: z.array(z.any()).optional(),
          images: z.array(z.any()).optional(),
        })
        .describe('Product fields to update'),
    }),
    httpMethod: 'PUT',
    path: '/com/products/{product_id}.json',
    scopes: ['com.write_products'],
  },
  {
    name: 'haravan_products_delete',
    project: 'products',
    description: 'Delete a product by ID.',
    schema: z.object({
      product_id: z.number().describe('Product ID to delete'),
    }),
    httpMethod: 'DELETE',
    path: '/com/products/{product_id}.json',
    scopes: ['com.write_products'],
  },
];

export const variantTools: McpTool[] = [
  {
    name: 'haravan_variants_list',
    project: 'products',
    description: 'List all variants of a product.',
    schema: z.object({
      product_id: z.number().describe('Product ID'),
      page: z.number().optional(),
      limit: z.number().optional(),
      fields: z.string().optional(),
    }),
    httpMethod: 'GET',
    path: '/com/products/{product_id}/variants.json',
    scopes: ['com.read_products'],
  },
  {
    name: 'haravan_variants_count',
    project: 'products',
    description: 'Count variants of a product.',
    schema: z.object({
      product_id: z.number().describe('Product ID'),
    }),
    httpMethod: 'GET',
    path: '/com/products/{product_id}/variants/count.json',
    scopes: ['com.read_products'],
  },
  {
    name: 'haravan_variants_get',
    project: 'products',
    description: 'Get a single variant by ID.',
    schema: z.object({
      variant_id: z.number().describe('Variant ID'),
      fields: z.string().optional(),
    }),
    httpMethod: 'GET',
    path: '/com/variants/{variant_id}.json',
    scopes: ['com.read_products'],
  },
  {
    name: 'haravan_variants_create',
    project: 'products',
    description: 'Create a new variant for a product.',
    schema: z.object({
      product_id: z.number().describe('Product ID'),
      variant: z
        .object({
          title: z.string().optional(),
          price: z.number().optional(),
          compare_at_price: z.number().optional(),
          sku: z.string().optional(),
          barcode: z.string().optional(),
          grams: z.number().optional(),
          inventory_management: z.enum(['haravan']).nullable().optional(),
          inventory_policy: z.enum(['deny', 'continue']).optional(),
          inventory_quantity: z.number().optional(),
          requires_shipping: z.boolean().optional(),
          taxable: z.boolean().optional(),
          option1: z.string().optional(),
          option2: z.string().optional(),
          option3: z.string().optional(),
          image_id: z.number().optional(),
        })
        .describe('Variant object'),
    }),
    httpMethod: 'POST',
    path: '/com/products/{product_id}/variants.json',
    scopes: ['com.write_products'],
  },
  {
    name: 'haravan_variants_update',
    project: 'products',
    description: 'Update an existing variant.',
    schema: z.object({
      variant_id: z.number().describe('Variant ID'),
      variant: z
        .object({
          id: z.number().describe('Variant ID'),
          price: z.number().optional(),
          compare_at_price: z.number().optional(),
          sku: z.string().optional(),
          barcode: z.string().optional(),
          grams: z.number().optional(),
          inventory_management: z.enum(['haravan']).nullable().optional(),
          inventory_policy: z.enum(['deny', 'continue']).optional(),
          requires_shipping: z.boolean().optional(),
          taxable: z.boolean().optional(),
          option1: z.string().optional(),
          option2: z.string().optional(),
          option3: z.string().optional(),
          image_id: z.number().optional(),
        })
        .describe('Variant fields to update'),
    }),
    httpMethod: 'PUT',
    path: '/com/variants/{variant_id}.json',
    scopes: ['com.write_products'],
  },
];
