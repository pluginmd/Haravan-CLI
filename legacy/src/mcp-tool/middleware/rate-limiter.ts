import { MiddlewareFn } from '../types';
import { logger } from '../../utils/logger';

/**
 * Haravan rate limiter middleware.
 * Implements leaky bucket: 80 request bucket, 4 requests/second leak rate.
 * Tracks usage from X-Haravan-Api-Call-Limit headers AND decays over time.
 */

const BUCKET_SIZE = 80;
const LEAK_RATE = 4; // requests per second
const SAFE_THRESHOLD = 70; // start slowing down at 70/80

let lastKnownUsage = 0;
let lastUpdatedAt = Date.now();

/**
 * Estimate current bucket usage by applying time-based decay.
 */
function estimateBucketUsage(): number {
  const elapsedMs = Date.now() - lastUpdatedAt;
  const leaked = (elapsedMs / 1000) * LEAK_RATE;
  return Math.max(0, lastKnownUsage - leaked);
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export const rateLimiterMiddleware: MiddlewareFn = async (ctx, next) => {
  const estimated = estimateBucketUsage();

  // If approaching limit, wait proportionally
  if (estimated >= SAFE_THRESHOLD) {
    const waitTime = Math.ceil(
      ((estimated - SAFE_THRESHOLD) / LEAK_RATE) * 1000
    );
    logger.debug(
      `Rate limiter: bucket ~${Math.round(estimated)}/${BUCKET_SIZE}, waiting ${waitTime}ms`
    );
    await sleep(waitTime);
  }

  const result = await next();

  // Update bucket usage from response metadata (authoritative source)
  if (ctx.meta.rateLimitUsed !== undefined) {
    lastKnownUsage = ctx.meta.rateLimitUsed;
    lastUpdatedAt = Date.now();
  }

  if (ctx.meta.retryAfter) {
    logger.warn(`Rate limited! Retry-After: ${ctx.meta.retryAfter}s`);
  }

  return result;
};

/** Reset state (for testing). */
export function _resetRateLimiter() {
  lastKnownUsage = 0;
  lastUpdatedAt = Date.now();
}
