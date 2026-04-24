import { composeMiddleware } from '../src/mcp-tool/middleware/chain';
import { errorHandlerMiddleware } from '../src/mcp-tool/middleware/error-handler';
import { MiddlewareContext, McpTool } from '../src/mcp-tool/types';
import { z } from 'zod';

const mockTool: McpTool = {
  name: 'test_tool',
  project: 'test',
  description: 'Test',
  schema: z.object({}),
  httpMethod: 'GET',
  path: '/test',
  scopes: ['com.read_shop'],
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

function axiosError(status: number, data?: any, headers?: Record<string, string>) {
  const err: any = new Error(`Request failed with status ${status}`);
  err.response = { status, data: data || {}, headers: headers || {} };
  return err;
}

describe('Error Handler Middleware', () => {
  test('should pass through successful results', async () => {
    const handler = jest.fn().mockResolvedValue({
      content: [{ type: 'text', text: '{"ok":true}' }],
    });
    const composed = composeMiddleware([errorHandlerMiddleware], handler);
    const result = await composed(createCtx());
    expect(result.isError).toBeUndefined();
  });

  test('should handle 401 Unauthorized', async () => {
    const handler = jest.fn().mockRejectedValue(axiosError(401));
    const composed = composeMiddleware([errorHandlerMiddleware], handler);
    const result = await composed(createCtx());
    expect(result.isError).toBe(true);
    expect(result.content[0].text).toContain('Authentication failed');
  });

  test('should handle 403 Forbidden with scope hint', async () => {
    const handler = jest.fn().mockRejectedValue(axiosError(403));
    const composed = composeMiddleware([errorHandlerMiddleware], handler);
    const result = await composed(createCtx());
    expect(result.isError).toBe(true);
    expect(result.content[0].text).toContain('com.read_shop');
  });

  test('should handle 404 Not Found', async () => {
    const handler = jest.fn().mockRejectedValue(axiosError(404));
    const composed = composeMiddleware([errorHandlerMiddleware], handler);
    const result = await composed(createCtx());
    expect(result.isError).toBe(true);
    expect(result.content[0].text).toContain('not found');
  });

  test('should handle 422 with validation details', async () => {
    const handler = jest.fn().mockRejectedValue(
      axiosError(422, { errors: { email: ['already taken'] } })
    );
    const composed = composeMiddleware([errorHandlerMiddleware], handler);
    const result = await composed(createCtx());
    expect(result.isError).toBe(true);
    expect(result.content[0].text).toContain('already taken');
  });

  test('should handle 429 rate limit with Retry-After', async () => {
    const ctx = createCtx();
    const handler = jest.fn().mockRejectedValue(
      axiosError(429, {}, { 'retry-after': '3.5' })
    );
    const composed = composeMiddleware([errorHandlerMiddleware], handler);
    const result = await composed(ctx);
    expect(result.isError).toBe(true);
    expect(result.content[0].text).toContain('3.5');
    expect(ctx.meta.retryAfter).toBe(3.5);
  });

  test('should handle 500 server error', async () => {
    const handler = jest.fn().mockRejectedValue(axiosError(500));
    const composed = composeMiddleware([errorHandlerMiddleware], handler);
    const result = await composed(createCtx());
    expect(result.isError).toBe(true);
    expect(result.content[0].text).toContain('server error');
  });

  test('should handle non-HTTP errors (no response)', async () => {
    const handler = jest.fn().mockRejectedValue(new Error('ECONNREFUSED'));
    const composed = composeMiddleware([errorHandlerMiddleware], handler);
    const result = await composed(createCtx());
    expect(result.isError).toBe(true);
    expect(result.content[0].text).toContain('ECONNREFUSED');
  });

  test('should not treat undefined status as >= 500', async () => {
    const handler = jest.fn().mockRejectedValue(new Error('timeout'));
    const composed = composeMiddleware([errorHandlerMiddleware], handler);
    const result = await composed(createCtx());
    const parsed = JSON.parse(result.content[0].text);
    expect(parsed.message).toBe('timeout');
    // Should NOT say "Haravan server error"
    expect(parsed.message).not.toContain('server error');
  });
});
