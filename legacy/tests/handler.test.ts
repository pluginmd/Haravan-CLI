import { haravanApiHandler } from '../src/mcp-tool/handler';
import { MiddlewareContext, McpTool } from '../src/mcp-tool/types';
import { z } from 'zod';

// Mock axios fully including interceptors
const mockRequest = jest.fn().mockResolvedValue({
  data: { customer: { id: 123 } },
  status: 200,
  headers: { 'x-haravan-api-call-limit': '5/80' },
});

jest.mock('axios', () => ({
  create: jest.fn(() => ({
    request: mockRequest,
    interceptors: {
      request: { use: jest.fn() },
      response: { use: jest.fn() },
    },
  })),
}));

const axios = require('axios');

function createTool(overrides: Partial<McpTool> = {}): McpTool {
  return {
    name: 'test_tool',
    project: 'test',
    description: 'Test',
    schema: z.object({}),
    httpMethod: 'GET',
    path: '/com/customers/{customer_id}.json',
    scopes: ['com.read_customers'],
    ...overrides,
  };
}

function createCtx(
  tool: McpTool,
  params: Record<string, any> = {}
): MiddlewareContext {
  return {
    tool,
    params,
    accessToken: 'test-token',
    domain: 'https://apis.haravan.com',
    webhookDomain: 'https://webhook.haravan.com',
    meta: {},
  };
}

describe('Haravan API Handler', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockRequest.mockResolvedValue({
      data: { customer: { id: 123 } },
      status: 200,
      headers: { 'x-haravan-api-call-limit': '5/80' },
    });
  });

  test('should resolve path parameters', async () => {
    const tool = createTool();
    const ctx = createCtx(tool, { customer_id: 123 });
    await haravanApiHandler(ctx);

    expect(mockRequest).toHaveBeenCalledWith(
      expect.objectContaining({
        method: 'GET',
        url: '/com/customers/123.json',
      })
    );
  });

  test('should put remaining GET params in query string', async () => {
    const tool = createTool();
    const ctx = createCtx(tool, { customer_id: 123, fields: 'id,email' });
    await haravanApiHandler(ctx);

    expect(mockRequest).toHaveBeenCalledWith(
      expect.objectContaining({
        params: { fields: 'id,email' },
      })
    );
  });

  test('should put POST params in request body', async () => {
    const tool = createTool({ httpMethod: 'POST', path: '/com/customers.json' });
    const ctx = createCtx(tool, { customer: { email: 'test@test.com' } });
    await haravanApiHandler(ctx);

    expect(mockRequest).toHaveBeenCalledWith(
      expect.objectContaining({
        data: { customer: { email: 'test@test.com' } },
      })
    );
  });

  test('should use webhook domain when isWebhook is true', async () => {
    const tool = createTool({ isWebhook: true, path: '/api/subscribe' });
    const ctx = createCtx(tool, {});
    await haravanApiHandler(ctx);

    expect(axios.create).toHaveBeenCalledWith(
      expect.objectContaining({
        baseURL: 'https://webhook.haravan.com',
      })
    );
  });

  test('should store rate limit info in meta', async () => {
    const tool = createTool();
    const ctx = createCtx(tool, { customer_id: 1 });
    await haravanApiHandler(ctx);

    expect(ctx.meta.rateLimitUsed).toBe(5);
    expect(ctx.meta.rateLimitMax).toBe(80);
  });

  test('should skip null/undefined params', async () => {
    const tool = createTool();
    const ctx = createCtx(tool, { customer_id: 1, fields: undefined, page: null });
    await haravanApiHandler(ctx);

    const callArgs = mockRequest.mock.calls[0][0];
    expect(callArgs.params).toBeUndefined();
  });
});
