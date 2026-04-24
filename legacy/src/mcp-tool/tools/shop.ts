import { z } from 'zod';
import { McpTool } from '../types';

export const shopTools: McpTool[] = [
  {
    name: 'haravan_shop_get',
    project: 'shop',
    description:
      'Get shop information: name, domain, email, currency, timezone, plan, address, checkout settings.',
    schema: z.object({}),
    httpMethod: 'GET',
    path: '/com/shop.json',
    scopes: ['com.read_shop'],
  },
  {
    name: 'haravan_locations_list',
    project: 'shop',
    description: 'List all locations/warehouses of the shop.',
    schema: z.object({}),
    httpMethod: 'GET',
    path: '/com/locations.json',
    scopes: ['com.read_shop'],
  },
  {
    name: 'haravan_locations_get',
    project: 'shop',
    description: 'Get a single location by ID.',
    schema: z.object({
      location_id: z.number().describe('Location ID'),
    }),
    httpMethod: 'GET',
    path: '/com/locations/{location_id}.json',
    scopes: ['com.read_shop'],
  },
  {
    name: 'haravan_users_list',
    project: 'shop',
    description: 'List all users/staff of the shop (Haravan Plus only).',
    schema: z.object({}),
    httpMethod: 'GET',
    path: '/com/users.json',
    scopes: ['com.read_shop'],
  },
  {
    name: 'haravan_users_get',
    project: 'shop',
    description: 'Get a single user by ID (Haravan Plus only).',
    schema: z.object({
      user_id: z.number().describe('User ID'),
    }),
    httpMethod: 'GET',
    path: '/com/users/{user_id}.json',
    scopes: ['com.read_shop'],
  },
  {
    name: 'haravan_shipping_rates_get',
    project: 'shop',
    description:
      'Get available shipping rates for a destination address. Pass address fields as flat query params.',
    schema: z.object({
      'shipping_address[address1]': z.string().optional().describe('Street address'),
      'shipping_address[city]': z.string().optional().describe('City'),
      'shipping_address[province]': z.string().optional().describe('Province'),
      'shipping_address[country]': z.string().optional().describe('Country'),
      'shipping_address[zip]': z.string().optional().describe('ZIP/Postal code'),
    }),
    httpMethod: 'GET',
    path: '/com/shipping_rates.json',
    scopes: ['com.read_orders'],
  },
];
