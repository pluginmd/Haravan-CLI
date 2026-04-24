import { MiddlewareContext, MiddlewareFn, CallToolResult, McpHandler } from '../types';

/**
 * Compose middleware functions into a single handler (Koa-style onion model).
 * Each middleware can modify ctx.params (request) and process the result (response).
 */
export function composeMiddleware(
  middlewares: MiddlewareFn[],
  handler: McpHandler
): McpHandler {
  return async (ctx: MiddlewareContext): Promise<CallToolResult> => {
    let index = -1;

    const dispatch = async (i: number): Promise<CallToolResult> => {
      if (i <= index) {
        throw new Error('next() called multiple times');
      }
      index = i;

      if (i === middlewares.length) {
        return handler(ctx);
      }

      return middlewares[i](ctx, () => dispatch(i + 1));
    };

    return dispatch(0);
  };
}
