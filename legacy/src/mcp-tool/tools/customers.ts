import { z } from 'zod';
import { McpTool } from '../types';

export const customerTools: McpTool[] = [
  {
    name: 'haravan_customers_list',
    project: 'customers',
    description:
      'List all customers. Supports pagination with page/limit params. Can filter by created_at_min, created_at_max, updated_at_min, updated_at_max, since_id, fields.',
    schema: z.object({
      page: z.number().optional().describe('Page number (default: 1)'),
      limit: z
        .number()
        .optional()
        .describe('Number of results per page (default: 50, max: 250)'),
      since_id: z
        .number()
        .optional()
        .describe('Return results after this ID'),
      created_at_min: z
        .string()
        .optional()
        .describe('Filter: created after this date (ISO 8601)'),
      created_at_max: z
        .string()
        .optional()
        .describe('Filter: created before this date (ISO 8601)'),
      updated_at_min: z
        .string()
        .optional()
        .describe('Filter: updated after this date (ISO 8601)'),
      updated_at_max: z
        .string()
        .optional()
        .describe('Filter: updated before this date (ISO 8601)'),
      fields: z
        .string()
        .optional()
        .describe('Comma-separated list of fields to include'),
      fetchAll: z
        .boolean()
        .optional()
        .describe('Auto-paginate to fetch all results (use with caution)'),
    }),
    httpMethod: 'GET',
    path: '/com/customers.json',
    scopes: ['com.read_customers'],
  },
  {
    name: 'haravan_customers_search',
    project: 'customers',
    description:
      'Search customers by query. Supports searching by email, phone, name, etc.',
    schema: z.object({
      query: z.string().describe('Search query string'),
      page: z.number().optional().describe('Page number'),
      limit: z.number().optional().describe('Results per page'),
      fields: z.string().optional().describe('Comma-separated fields'),
    }),
    httpMethod: 'GET',
    path: '/com/customers/search.json',
    scopes: ['com.read_customers'],
  },
  {
    name: 'haravan_customers_count',
    project: 'customers',
    description: 'Get total count of customers.',
    schema: z.object({
      created_at_min: z.string().optional().describe('Created after (ISO 8601)'),
      created_at_max: z.string().optional().describe('Created before (ISO 8601)'),
      updated_at_min: z.string().optional().describe('Updated after (ISO 8601)'),
      updated_at_max: z.string().optional().describe('Updated before (ISO 8601)'),
    }),
    httpMethod: 'GET',
    path: '/com/customers/count.json',
    scopes: ['com.read_customers'],
  },
  {
    name: 'haravan_customers_get',
    project: 'customers',
    description: 'Get a single customer by ID.',
    schema: z.object({
      customer_id: z.number().describe('Customer ID'),
      fields: z.string().optional().describe('Comma-separated fields'),
    }),
    httpMethod: 'GET',
    path: '/com/customers/{customer_id}.json',
    scopes: ['com.read_customers'],
  },
  {
    name: 'haravan_customers_create',
    project: 'customers',
    description:
      'Create a new customer. Email or phone is required. Supports addresses, tags, marketing consent.',
    schema: z.object({
      customer: z
        .object({
          first_name: z.string().optional().describe('First name'),
          last_name: z.string().optional().describe('Last name'),
          email: z.string().optional().describe('Email address (unique)'),
          phone: z.string().optional().describe('Phone number (unique)'),
          accepts_marketing: z
            .boolean()
            .optional()
            .describe('Accepts email marketing'),
          tags: z.string().optional().describe('Comma-separated tags'),
          note: z.string().optional().describe('Note about the customer'),
          birthday: z.string().optional().describe('Birthday (ISO 8601)'),
          gender: z
            .number()
            .optional()
            .describe('Gender: null=not set, 1=male, 0=female'),
          addresses: z
            .array(
              z.object({
                address1: z.string().optional(),
                address2: z.string().optional(),
                city: z.string().optional(),
                province: z.string().optional(),
                country: z.string().optional(),
                phone: z.string().optional(),
                zip: z.string().optional(),
                first_name: z.string().optional(),
                last_name: z.string().optional(),
                company: z.string().optional(),
                district: z.string().optional(),
                ward: z.string().optional(),
              })
            )
            .optional()
            .describe('Customer addresses'),
          send_email_invite: z
            .boolean()
            .optional()
            .describe('Send account invitation email'),
          send_email_welcome: z
            .boolean()
            .optional()
            .describe('Send welcome email'),
        })
        .describe('Customer object'),
    }),
    httpMethod: 'POST',
    path: '/com/customers.json',
    scopes: ['com.write_customers'],
  },
  {
    name: 'haravan_customers_update',
    project: 'customers',
    description: 'Update an existing customer.',
    schema: z.object({
      customer_id: z.number().describe('Customer ID to update'),
      customer: z
        .object({
          id: z.number().describe('Customer ID'),
          first_name: z.string().optional(),
          last_name: z.string().optional(),
          email: z.string().optional(),
          phone: z.string().optional(),
          accepts_marketing: z.boolean().optional(),
          tags: z.string().optional(),
          note: z.string().optional(),
          birthday: z.string().optional(),
          gender: z.number().optional(),
        })
        .describe('Customer fields to update'),
    }),
    httpMethod: 'PUT',
    path: '/com/customers/{customer_id}.json',
    scopes: ['com.write_customers'],
  },
  {
    name: 'haravan_customers_delete',
    project: 'customers',
    description: 'Delete a customer by ID.',
    schema: z.object({
      customer_id: z.number().describe('Customer ID to delete'),
    }),
    httpMethod: 'DELETE',
    path: '/com/customers/{customer_id}.json',
    scopes: ['com.write_customers'],
  },
  {
    name: 'haravan_customers_groups',
    project: 'customers',
    description: 'Get all customer groups.',
    schema: z.object({}),
    httpMethod: 'GET',
    path: '/com/customers/groups.json',
    scopes: ['com.read_customers'],
  },
];

export const customerAddressTools: McpTool[] = [
  {
    name: 'haravan_customer_addresses_list',
    project: 'customers',
    description: 'List all addresses of a customer.',
    schema: z.object({
      customer_id: z.number().describe('Customer ID'),
      page: z.number().optional(),
      limit: z.number().optional(),
    }),
    httpMethod: 'GET',
    path: '/com/customers/{customer_id}/addresses.json',
    scopes: ['com.read_customers'],
  },
  {
    name: 'haravan_customer_addresses_get',
    project: 'customers',
    description: 'Get a specific address of a customer.',
    schema: z.object({
      customer_id: z.number().describe('Customer ID'),
      address_id: z.number().describe('Address ID'),
    }),
    httpMethod: 'GET',
    path: '/com/customers/{customer_id}/addresses/{address_id}.json',
    scopes: ['com.read_customers'],
  },
  {
    name: 'haravan_customer_addresses_create',
    project: 'customers',
    description: 'Create a new address for a customer.',
    schema: z.object({
      customer_id: z.number().describe('Customer ID'),
      address: z
        .object({
          address1: z.string().optional(),
          address2: z.string().optional(),
          city: z.string().optional(),
          province: z.string().optional(),
          province_code: z.string().optional(),
          country: z.string().optional(),
          country_code: z.string().optional(),
          district: z.string().optional(),
          district_code: z.string().optional(),
          ward: z.string().optional(),
          ward_code: z.string().optional(),
          phone: z.string().optional(),
          zip: z.string().optional(),
          first_name: z.string().optional(),
          last_name: z.string().optional(),
          company: z.string().optional(),
        })
        .describe('Address object'),
    }),
    httpMethod: 'POST',
    path: '/com/customers/{customer_id}/addresses.json',
    scopes: ['com.write_customers'],
  },
  {
    name: 'haravan_customer_addresses_update',
    project: 'customers',
    description: 'Update a customer address.',
    schema: z.object({
      customer_id: z.number().describe('Customer ID'),
      address_id: z.number().describe('Address ID'),
      address: z
        .object({
          address1: z.string().optional(),
          address2: z.string().optional(),
          city: z.string().optional(),
          province: z.string().optional(),
          district: z.string().optional(),
          ward: z.string().optional(),
          phone: z.string().optional(),
          zip: z.string().optional(),
          first_name: z.string().optional(),
          last_name: z.string().optional(),
          company: z.string().optional(),
        })
        .describe('Address fields to update'),
    }),
    httpMethod: 'PUT',
    path: '/com/customers/{customer_id}/addresses/{address_id}.json',
    scopes: ['com.write_customers'],
  },
  {
    name: 'haravan_customer_addresses_delete',
    project: 'customers',
    description: 'Delete a customer address.',
    schema: z.object({
      customer_id: z.number().describe('Customer ID'),
      address_id: z.number().describe('Address ID'),
    }),
    httpMethod: 'DELETE',
    path: '/com/customers/{customer_id}/addresses/{address_id}.json',
    scopes: ['com.write_customers'],
  },
  {
    name: 'haravan_customer_addresses_set_default',
    project: 'customers',
    description: 'Set a customer address as default.',
    schema: z.object({
      customer_id: z.number().describe('Customer ID'),
      address_id: z.number().describe('Address ID to set as default'),
    }),
    httpMethod: 'PUT',
    path: '/com/customers/{customer_id}/addresses/{address_id}/default.json',
    scopes: ['com.write_customers'],
  },
];
