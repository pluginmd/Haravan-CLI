import { z } from 'zod';

export interface McpTool {
  /** Tool name, e.g., "haravan_customers_list" */
  name: string;
  /** Category/project, e.g., "customers", "orders", "products" */
  project: string;
  /** Human-readable description */
  description: string;
  /** Zod schema for input parameters */
  schema: z.ZodObject<any>;
  /** HTTP method: GET, POST, PUT, DELETE */
  httpMethod: string;
  /** API endpoint path, e.g., "/com/customers.json" */
  path: string;
  /** Required scopes, e.g., ["com.read_customers"] */
  scopes: string[];
  /** Whether this is a webhook API (uses webhook domain) */
  isWebhook?: boolean;
  /** Custom handler override */
  customHandler?: McpHandler;
}

export type McpHandler = (
  ctx: MiddlewareContext
) => Promise<CallToolResult>;

export interface MiddlewareContext {
  tool: McpTool;
  params: Record<string, any>;
  accessToken: string;
  domain: string;
  webhookDomain: string;
  meta: Record<string, any>;
}

export type MiddlewareFn = (
  ctx: MiddlewareContext,
  next: () => Promise<CallToolResult>
) => Promise<CallToolResult>;

export interface CallToolResult {
  content: Array<{ type: 'text'; text: string }>;
  isError?: boolean;
}

export interface ToolPreset {
  name: string;
  description: string;
  tools: string[];
}
