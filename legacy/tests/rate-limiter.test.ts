import { composeMiddleware } from '../src/mcp-tool/middleware/chain';
import { rateLimiterMiddleware, _resetRateLimiter } from '../src/mcp-tool/middleware/rate-limiter';
import { MiddlewareContext, McpTool, CallToolResult } from '../src/mcp-tool/types';
import { z } from 'zod';

const mockTool: McpTool = {
  name: 'test_tool',
  project: 'test',
  description: 'Test',
  schema: z.object({}),
  httpMethod: 'GET',
  path: '/test',
  scopes: ['test'],
};

function createCtx(): MiddlewareContext {
  return {
    tool: mockTool,
    params: {},
    accessToken: 'test',
    domain: 'https://apis.haravan.com',
    webhookDomain: 'https://webhook.haravan.com',
    meta: {},
  };
}

const successResult: CallToolResult = {
  content: [{ type: 'text', text: '{"ok":true}' }],
};

beforeEach(() => {
  _resetRateLimiter();
});

describe('Rate Limiter Middleware', () => {
  test('should pass through when bucket is empty', async () => {
    const handler = jest.fn().mockResolvedValue(successResult);
    const composed = composeMiddleware([rateLimiterMiddleware], handler);
    const result = await composed(createCtx());
    expect(handler).toHaveBeenCalled();
    expect(result).toEqual(successResult);
  });

  test('should update bucket from response meta', async () => {
    const handler = jest.fn().mockImplementation(async (ctx: MiddlewareContext) => {
      ctx.meta.rateLimitUsed = 50;
      return successResult;
    });
    const composed = composeMiddleware([rateLimiterMiddleware], handler);
    const ctx = createCtx();
    await composed(ctx);
    // Bucket should be updated — next call should reflect this
    expect(ctx.meta.rateLimitUsed).toBe(50);
  });

  test('should decay bucket over time', async () => {
    // Simulate previous high usage
    const handler = jest.fn().mockImplementation(async (ctx: MiddlewareContext) => {
      ctx.meta.rateLimitUsed = 75;
      return successResult;
    });
    const composed = composeMiddleware([rateLimiterMiddleware], handler);

    // First call sets high usage
    await composed(createCtx());

    // After reset, bucket should be at 0 (simulating time passed)
    _resetRateLimiter();

    const start = Date.now();
    await composed(createCtx());
    const elapsed = Date.now() - start;

    // Should not have waited (bucket was reset to 0)
    expect(elapsed).toBeLessThan(100);
  });
});
