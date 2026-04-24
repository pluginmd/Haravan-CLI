import { MiddlewareFn } from '../types';
import { HaravanClient } from '../../utils/http-client';
import { logger } from '../../utils/logger';

/**
 * Auto-pagination middleware.
 * For GET list endpoints, automatically paginates if fetchAll=true.
 * Instead of re-calling next() (which breaks the middleware guard),
 * we call the Haravan API directly for subsequent pages.
 */

const MAX_PAGES = 20;
const MAX_RECORDS = 5000;

export const paginationMiddleware: MiddlewareFn = async (ctx, next) => {
  const { fetchAll, ...restParams } = ctx.params;

  // Only auto-paginate for GET list operations when explicitly requested
  if (!fetchAll || ctx.tool.httpMethod !== 'GET') {
    ctx.params = restParams;
    return next();
  }

  ctx.params = restParams;

  // Set initial pagination params (Haravan API max is 50 per page)
  if (!ctx.params.limit) {
    ctx.params.limit = 50;
  }
  const startPage = ctx.params.page || 1;
  ctx.params.page = startPage;

  // First page goes through the normal middleware chain
  const firstResult = await next();

  let resourceKey: string | null = null;
  let firstItems: any[];

  try {
    const parsed = JSON.parse(firstResult.content[0].text);
    resourceKey =
      Object.keys(parsed).find((k) => Array.isArray(parsed[k])) || null;

    if (!resourceKey || !Array.isArray(parsed[resourceKey])) {
      return firstResult;
    }
    firstItems = parsed[resourceKey];
  } catch {
    return firstResult;
  }

  const allResults: any[] = [...firstItems];

  // If first page is already short, no more pages
  if (firstItems.length < ctx.params.limit) {
    return firstResult;
  }

  // Fetch subsequent pages directly (bypass middleware chain)
  const client = new HaravanClient({
    domain: ctx.tool.isWebhook ? ctx.webhookDomain : ctx.domain,
    accessToken: ctx.accessToken,
  });

  let page = startPage + 1;
  for (let i = 1; i < MAX_PAGES; i++) {
    // Build query params
    const queryParams: Record<string, any> = {};
    for (const [key, value] of Object.entries(ctx.params)) {
      if (value !== undefined && value !== null) {
        queryParams[key] = value;
      }
    }
    queryParams.page = page;

    try {
      const response = await client.get(ctx.tool.path, queryParams);
      const data = response.data;

      if (
        resourceKey &&
        data[resourceKey] &&
        Array.isArray(data[resourceKey])
      ) {
        const items = data[resourceKey];
        allResults.push(...items);

        logger.debug(
          `Pagination: page ${page}, got ${items.length} items, total ${allResults.length}`
        );

        if (items.length < ctx.params.limit) break;
        if (allResults.length >= MAX_RECORDS) {
          logger.warn(`Pagination: hit max records limit (${MAX_RECORDS})`);
          break;
        }

        page++;
      } else {
        break;
      }
    } catch (err) {
      logger.error(`Pagination error on page ${page}:`, err);
      break;
    }
  }

  const responseObj: Record<string, any> = {};
  responseObj[resourceKey!] = allResults;
  responseObj._pagination = {
    total_fetched: allResults.length,
    pages_fetched: page - startPage + 1,
    truncated: allResults.length >= MAX_RECORDS,
  };

  return {
    content: [
      {
        type: 'text' as const,
        text: JSON.stringify(responseObj, null, 2),
      },
    ],
  };
};
