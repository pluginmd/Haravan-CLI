import { composeMiddleware } from '../src/mcp-tool/middleware/chain';
import { validationMiddleware } from '../src/mcp-tool/middleware/validation';
import { MiddlewareContext, CallToolResult, McpTool, MiddlewareFn } from '../src/mcp-tool/types';
import { z } from 'zod';

const mockTool: McpTool = {
  name: 'test_tool',
  project: 'test',
  description: 'Test tool',
  schema: z.object({ name: z.string() }),
  httpMethod: 'GET',
  path: '/test',
  scopes: ['test'],
};

function createCtx(params: Record<string, any> = {}): MiddlewareContext {
  return {
    tool: mockTool,
    params,
    accessToken: 'test-token',
    domain: 'https://apis.haravan.com',
    webhookDomain: 'https://webhook.haravan.com',
    meta: {},
  };
}

const successResult: CallToolResult = {
  content: [{ type: 'text', text: '{"ok": true}' }],
};

describe('Middleware Chain', () => {
  test('should execute handler when no middleware', async () => {
    const handler = jest.fn().mockResolvedValue(successResult);
    const composed = composeMiddleware([], handler);
    const ctx = createCtx();
    const result = await composed(ctx);
    expect(handler).toHaveBeenCalledWith(ctx);
    expect(result).toEqual(successResult);
  });

  test('should execute middleware in order (onion model)', async () => {
    const order: string[] = [];

    const mw1: MiddlewareFn = async (ctx, next) => {
      order.push('mw1-before');
      const result = await next();
      order.push('mw1-after');
      return result;
    };

    const mw2: MiddlewareFn = async (ctx, next) => {
      order.push('mw2-before');
      const result = await next();
      order.push('mw2-after');
      return result;
    };

    const handler = jest.fn().mockImplementation(async () => {
      order.push('handler');
      return successResult;
    });

    const composed = composeMiddleware([mw1, mw2], handler);
    await composed(createCtx());

    expect(order).toEqual([
      'mw1-before',
      'mw2-before',
      'handler',
      'mw2-after',
      'mw1-after',
    ]);
  });

  test('middleware can modify params', async () => {
    const mw: MiddlewareFn = async (ctx, next) => {
      ctx.params.added = true;
      return next();
    };

    const handler = jest.fn().mockResolvedValue(successResult);
    const composed = composeMiddleware([mw], handler);
    const ctx = createCtx({ original: true });
    await composed(ctx);

    expect(ctx.params.added).toBe(true);
    expect(ctx.params.original).toBe(true);
  });

  test('middleware can short-circuit', async () => {
    const errorResult: CallToolResult = {
      content: [{ type: 'text', text: 'blocked' }],
      isError: true,
    };

    const mw: MiddlewareFn = async (_ctx, _next) => {
      return errorResult; // Don't call next()
    };

    const handler = jest.fn().mockResolvedValue(successResult);
    const composed = composeMiddleware([mw], handler);
    const result = await composed(createCtx());

    expect(handler).not.toHaveBeenCalled();
    expect(result.isError).toBe(true);
  });
});

describe('Validation Middleware', () => {
  test('should pass clean input', async () => {
    const handler = jest.fn().mockResolvedValue(successResult);
    const composed = composeMiddleware([validationMiddleware], handler);
    const ctx = createCtx({ name: 'Hello World' });
    const result = await composed(ctx);
    expect(handler).toHaveBeenCalled();
    expect(result.isError).toBeUndefined();
  });

  test('should reject control characters', async () => {
    const handler = jest.fn().mockResolvedValue(successResult);
    const composed = composeMiddleware([validationMiddleware], handler);
    const ctx = createCtx({ name: 'Hello\x00World' });
    const result = await composed(ctx);
    expect(handler).not.toHaveBeenCalled();
    expect(result.isError).toBe(true);
  });
});
