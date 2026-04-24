import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { logger } from '../../utils/logger';

/**
 * Start MCP server with stdio transport.
 * This is the primary transport mode for CLI-based AI tools (Claude, Cursor, etc.)
 */
export async function startStdioTransport(mcpServer: McpServer): Promise<void> {
  const transport = new StdioServerTransport();
  await mcpServer.connect(transport);
  logger.info('Haravan MCP server started (stdio transport)');
}
