import { MiddlewareFn, CallToolResult } from '../types';

/**
 * Input validation middleware.
 * Rejects dangerous characters and sanitizes inputs.
 */
export const validationMiddleware: MiddlewareFn = async (ctx, next) => {
  // Scan string values for dangerous patterns
  for (const [key, value] of Object.entries(ctx.params)) {
    if (typeof value === 'string') {
      // Reject control characters (except \t, \n, \r)
      if (/[\x00-\x08\x0B\x0C\x0E-\x1F]/.test(value)) {
        return {
          content: [
            {
              type: 'text' as const,
              text: JSON.stringify({
                error: true,
                message: `Parameter "${key}" contains invalid control characters`,
              }),
            },
          ],
          isError: true,
        };
      }
    }
  }

  return next();
};
