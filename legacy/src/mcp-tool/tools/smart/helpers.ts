import { HaravanClient, HaravanApiResponse } from '../../../utils/http-client';
import { MiddlewareContext } from '../../types';
import { logger } from '../../../utils/logger';

/**
 * Create a HaravanClient from middleware context.
 */
export function clientFromCtx(ctx: MiddlewareContext): HaravanClient {
  return new HaravanClient({
    domain: ctx.domain,
    accessToken: ctx.accessToken,
  });
}

/**
 * Fetch ALL records from a paginated Haravan API endpoint.
 * Uses since_id for efficient pagination when possible.
 * Returns the aggregated array + metadata.
 */
export async function fetchAll<T = any>(
  client: HaravanClient,
  path: string,
  resourceKey: string,
  params: Record<string, any> = {},
  options: { maxPages?: number; fields?: string } = {}
): Promise<{ items: T[]; apiCalls: number }> {
  const maxPages = options.maxPages || 100;
  const PAGE_SIZE = 50; // Haravan API max per page
  const allItems: T[] = [];
  let page = 1;
  let apiCalls = 0;

  const queryParams: Record<string, any> = {
    limit: PAGE_SIZE,
    ...params,
  };

  if (options.fields) {
    queryParams.fields = options.fields;
  }

  for (let i = 0; i < maxPages; i++) {
    queryParams.page = page;
    apiCalls++;

    const response = await client.get<any>(path, queryParams);
    const items = response.data?.[resourceKey];

    if (!items || !Array.isArray(items) || items.length === 0) {
      break;
    }

    allItems.push(...items);
    logger.debug(`fetchAll ${path}: page ${page}, got ${items.length}, total ${allItems.length}`);

    if (items.length < PAGE_SIZE) break;

    // Throttle to respect rate limit (leaky bucket 4 req/s)
    if (response.rateLimitUsed && response.rateLimitUsed > 60) {
      await sleep(1000);
    } else {
      await sleep(250);
    }

    page++;
  }

  return { items: allItems, apiCalls };
}

/**
 * Fetch count from a count endpoint.
 */
export async function fetchCount(
  client: HaravanClient,
  path: string,
  params: Record<string, any> = {}
): Promise<number> {
  const response = await client.get<any>(path, params);
  return response.data?.count ?? 0;
}

/**
 * Build a successful tool result with JSON data.
 */
export function ok(data: any) {
  return {
    content: [{ type: 'text' as const, text: JSON.stringify(data, null, 2) }],
  };
}

/**
 * Build an error tool result.
 */
export function err(message: string) {
  return {
    content: [{ type: 'text' as const, text: JSON.stringify({ error: true, message }) }],
    isError: true as const,
  };
}

export function sleep(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

/**
 * Parse a date string or default to N days ago.
 */
export function parseDate(input: string | undefined, defaultDaysAgo: number): string {
  if (input) return input;
  const d = new Date();
  d.setDate(d.getDate() - defaultDaysAgo);
  return d.toISOString();
}

/**
 * Strip null/undefined/empty values from an object to save tokens.
 */
export function stripEmpty(obj: any): any {
  if (obj === null || obj === undefined) return undefined;
  if (Array.isArray(obj)) return obj.map(stripEmpty).filter((v: any) => v !== undefined);
  if (typeof obj === 'object') {
    const result: Record<string, any> = {};
    for (const [k, v] of Object.entries(obj)) {
      const stripped = stripEmpty(v);
      if (stripped !== undefined && stripped !== '' && stripped !== null) {
        if (Array.isArray(stripped) && stripped.length === 0) continue;
        if (typeof stripped === 'object' && !Array.isArray(stripped) && Object.keys(stripped).length === 0) continue;
        result[k] = stripped;
      }
    }
    return Object.keys(result).length > 0 ? result : undefined;
  }
  return obj;
}
