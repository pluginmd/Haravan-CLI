import { StreamableHTTPServerTransport } from '@modelcontextprotocol/sdk/server/streamableHttp.js';
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import express from 'express';
import { logger } from '../../utils/logger';

/**
 * Start MCP server with Streamable HTTP transport.
 * Each session gets its own transport and server instance.
 */
export async function startHttpTransport(
  createServer: () => McpServer,
  port: number
): Promise<void> {
  const app = express();
  app.use(express.json());

  const sessions = new Map<string, { transport: StreamableHTTPServerTransport; server: McpServer }>();

  app.post('/mcp', async (req, res) => {
    const sessionId = req.headers['mcp-session-id'] as string | undefined;

    if (sessionId && sessions.has(sessionId)) {
      const session = sessions.get(sessionId)!;
      await session.transport.handleRequest(req, res, req.body);
      return;
    }

    // New session
    const transport = new StreamableHTTPServerTransport({
      sessionIdGenerator: () => crypto.randomUUID(),
    });
    const server = createServer();

    await server.connect(transport);
    await transport.handleRequest(req, res, req.body);

    // Store session AFTER handleRequest (sessionId is now set in response headers)
    const newSessionId = res.getHeader('mcp-session-id') as string;
    if (newSessionId) {
      sessions.set(newSessionId, { transport, server });
      logger.info(`New HTTP session: ${newSessionId}`);
    }

    // Cleanup on close
    transport.onclose = () => {
      if (newSessionId) sessions.delete(newSessionId);
      logger.info(`HTTP session closed: ${newSessionId}`);
    };
  });

  app.get('/mcp', async (req, res) => {
    const sessionId = req.headers['mcp-session-id'] as string | undefined;
    if (sessionId && sessions.has(sessionId)) {
      const session = sessions.get(sessionId)!;
      await session.transport.handleRequest(req, res, req.body);
      return;
    }
    res.status(400).json({ error: 'No session. Send POST /mcp first.' });
  });

  app.delete('/mcp', async (req, res) => {
    const sessionId = req.headers['mcp-session-id'] as string | undefined;
    if (sessionId && sessions.has(sessionId)) {
      const session = sessions.get(sessionId)!;
      await session.transport.handleRequest(req, res, req.body);
      sessions.delete(sessionId);
      return;
    }
    res.status(400).json({ error: 'Unknown session.' });
  });

  app.get('/health', (_req, res) => {
    res.json({ status: 'ok' });
  });

  app.listen(port, () => {
    logger.info(`Haravan MCP server started (HTTP streamable transport on port ${port})`);
  });
}
