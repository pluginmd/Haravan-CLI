/**
 * Integration test: boots the actual MCP server and verifies
 * tool listing works end-to-end via the MCP SDK client.
 */
import { initHaravanMcpServer } from '../src/mcp-server/init';
import { mergeConfig } from '../src/utils/config';
import { InMemoryTransport } from '@modelcontextprotocol/sdk/inMemory.js';
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { setLogLevel, LogLevel } from '../src/utils/logger';

// Suppress logs during tests
setLogLevel(LogLevel.SILENT);

describe('MCP Server Integration', () => {
  let client: Client;
  let cleanup: () => Promise<void>;

  beforeAll(async () => {
    const config = mergeConfig({
      accessToken: 'test-token-for-integration',
      tools: ['preset.default'],
    });

    const { mcpServer } = initHaravanMcpServer(config);

    // Create in-memory transport pair
    const [clientTransport, serverTransport] = InMemoryTransport.createLinkedPair();

    // Connect server
    await mcpServer.connect(serverTransport);

    // Connect client
    client = new Client({ name: 'test-client', version: '1.0.0' });
    await client.connect(clientTransport);

    cleanup = async () => {
      await client.close();
      await mcpServer.close();
    };
  });

  afterAll(async () => {
    if (cleanup) await cleanup();
  });

  test('should list tools via MCP protocol', async () => {
    const result = await client.listTools();
    expect(result.tools.length).toBe(12); // preset.default = 5 smart + 7 detail
  });

  test('should have correct tool names (smart + detail)', async () => {
    const result = await client.listTools();
    const toolNames = result.tools.map((t: any) => t.name);

    // Smart aggregation tools
    expect(toolNames).toContain('hrv_orders_summary');
    expect(toolNames).toContain('hrv_top_products');
    expect(toolNames).toContain('hrv_customer_segments');
    // Detail tools
    expect(toolNames).toContain('haravan_shop_get');
    expect(toolNames).toContain('haravan_customers_search');
  });

  test('each tool should have description and inputSchema', async () => {
    const result = await client.listTools();
    for (const tool of result.tools) {
      expect(tool.name).toBeTruthy();
      expect(tool.description).toBeTruthy();
      expect(tool.inputSchema).toBeTruthy();
      expect(tool.inputSchema.type).toBe('object');
    }
  });

  test('should call tool and get error for fake token', async () => {
    // This will fail because the token is fake, but we verify
    // the tool call goes through the middleware chain correctly
    const result = await client.callTool({
      name: 'haravan_shop_get',
      arguments: {},
    });

    // Should return error (network/auth error) but not crash
    expect(result.content).toBeTruthy();
    expect(Array.isArray(result.content)).toBe(true);
    expect(result.isError).toBe(true);
  });

  test('should list all 78 tools with "all" filter', async () => {
    const config = mergeConfig({
      accessToken: 'test-token',
      tools: ['all'],
    });

    const { mcpServer: allServer } = initHaravanMcpServer(config);
    const [ct, st] = InMemoryTransport.createLinkedPair();
    await allServer.connect(st);

    const allClient = new Client({ name: 'test-all', version: '1.0.0' });
    await allClient.connect(ct);

    const result = await allClient.listTools();
    expect(result.tools.length).toBe(78); // 15 smart + 63 raw

    await allClient.close();
    await allServer.close();
  });

  test('should list only smart tools with preset.smart', async () => {
    const config = mergeConfig({
      accessToken: 'test-token',
      tools: ['preset.smart'],
    });

    const { mcpServer: smartServer } = initHaravanMcpServer(config);
    const [ct, st] = InMemoryTransport.createLinkedPair();
    await smartServer.connect(st);

    const smartClient = new Client({ name: 'test-smart', version: '1.0.0' });
    await smartClient.connect(ct);

    const result = await smartClient.listTools();
    expect(result.tools.length).toBe(15);

    // All should be hrv_* smart tools
    for (const tool of result.tools) {
      expect(tool.name).toMatch(/^hrv_/);
    }

    await smartClient.close();
    await smartServer.close();
  });
});
