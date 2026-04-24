import { HaravanClient } from '../utils/http-client';
import { MiddlewareContext, CallToolResult, McpTool } from './types';
import { logger } from '../utils/logger';

/**
 * Default handler that calls Haravan API endpoints.
 * Resolves path parameters from ctx.params and makes the HTTP request.
 */
export async function haravanApiHandler(
  ctx: MiddlewareContext
): Promise<CallToolResult> {
  const { tool, params, accessToken, domain, webhookDomain } = ctx;

  const client = new HaravanClient({
    domain: tool.isWebhook ? webhookDomain : domain,
    accessToken,
  });

  // Resolve path parameters (e.g., /com/customers/{customer_id}.json)
  let resolvedPath = tool.path;
  const queryParams: Record<string, any> = {};
  const bodyParams: Record<string, any> = {};

  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null) continue;

    // Check if this is a path parameter
    const pathPlaceholder = `{${key}}`;
    if (resolvedPath.includes(pathPlaceholder)) {
      resolvedPath = resolvedPath.replace(pathPlaceholder, String(value));
    } else if (tool.httpMethod === 'GET' || tool.httpMethod === 'DELETE') {
      // For GET/DELETE, remaining params go to query string
      queryParams[key] = value;
    } else {
      // For POST/PUT, remaining params go to request body
      bodyParams[key] = value;
    }
  }

  logger.debug(`${tool.httpMethod} ${resolvedPath}`, { queryParams, bodyParams });

  const response = await client.request(
    tool.httpMethod,
    resolvedPath,
    Object.keys(bodyParams).length > 0 ? bodyParams : undefined,
    Object.keys(queryParams).length > 0 ? queryParams : undefined
  );

  // Store rate limit info in meta for middleware
  ctx.meta.rateLimitUsed = response.rateLimitUsed;
  ctx.meta.rateLimitMax = response.rateLimitMax;
  ctx.meta.retryAfter = response.retryAfter;

  return {
    content: [
      {
        type: 'text' as const,
        text: JSON.stringify(response.data, null, 2),
      },
    ],
  };
}
