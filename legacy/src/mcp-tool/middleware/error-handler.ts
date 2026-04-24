import { MiddlewareFn, CallToolResult } from '../types';
import { logger } from '../../utils/logger';

/**
 * Error handler middleware.
 * Catches API errors and formats them as MCP error responses.
 */
export const errorHandlerMiddleware: MiddlewareFn = async (ctx, next) => {
  try {
    return await next();
  } catch (error: any) {
    const status = error.response?.status;
    const data = error.response?.data;

    let message: string;

    if (status === 401) {
      message = 'Authentication failed. Check your access token.';
    } else if (status === 403) {
      message = `Forbidden. Your app may not have the required scope: ${ctx.tool.scopes.join(', ')}`;
    } else if (status === 404) {
      message = `Resource not found: ${ctx.tool.path}`;
    } else if (status === 422) {
      message = `Validation error: ${JSON.stringify(data?.errors || data)}`;
    } else if (status === 429) {
      const retryAfter = error.response?.headers?.['retry-after'] || '2';
      message = `Rate limited. Retry after ${retryAfter} seconds.`;
      ctx.meta.retryAfter = parseFloat(retryAfter);
    } else if (typeof status === 'number' && status >= 500) {
      message = `Haravan server error (${status}). Try again later.`;
    } else {
      message = error.message || 'Unknown error occurred';
    }

    logger.error(`Tool ${ctx.tool.name} error:`, message);

    return {
      content: [
        {
          type: 'text' as const,
          text: JSON.stringify({
            error: true,
            status,
            message,
            details: data,
          }),
        },
      ],
      isError: true,
    };
  }
};
