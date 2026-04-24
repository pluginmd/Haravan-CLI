#!/usr/bin/env node

import { Command } from 'commander';
import * as dotenv from 'dotenv';
import { initHaravanMcpServer } from './mcp-server/init';
import { startStdioTransport } from './mcp-server/transport/stdio';
import { startHttpTransport } from './mcp-server/transport/sse';
import { performOAuthLogin } from './auth/oauth';
import { loadToken, loadAllTokens, removeToken, isTokenExpired } from './auth/token-store';
import { mergeConfig } from './utils/config';
import { setLogLevel, LogLevel, logger } from './utils/logger';
import { resolveToolFilter } from './mcp-tool/presets';
import { allTools, PRESETS } from './mcp-tool/registry';

dotenv.config();

const VERSION = require('../package.json').version as string;

const program = new Command();

program
  .name('haravan-mcp')
  .description(
    'Haravan E-commerce MCP Server - Connect AI assistants to Haravan stores'
  )
  .version(VERSION);

// ============================================
// MCP Command - Start MCP Server
// ============================================
program
  .command('mcp')
  .description('Start the Haravan MCP server')
  .option('-t, --token <token>', 'Haravan access token (private app)')
  .option('-a, --app-id <appId>', 'Haravan App ID (for OAuth)')
  .option('-s, --app-secret <appSecret>', 'Haravan App Secret (for OAuth)')
  .option(
    '-d, --domain <domain>',
    'API domain',
    'https://apis.haravan.com'
  )
  .option(
    '--webhook-domain <webhookDomain>',
    'Webhook domain',
    'https://webhook.haravan.com'
  )
  .option(
    '--tools <tools>',
    'Comma-separated list of tools/presets to enable (e.g., "preset.default,products,haravan_webhooks_list")'
  )
  .option(
    '-m, --mode <mode>',
    'Transport mode: stdio, http',
    'stdio'
  )
  .option('-p, --port <port>', 'Port for HTTP mode', '3000')
  .option('--oauth', 'Use OAuth (requires stored token from login)')
  .option('--debug', 'Enable debug logging')
  .action(async (options) => {
    if (options.debug) {
      setLogLevel(LogLevel.DEBUG);
    }

    // Resolve access token
    let accessToken = options.token || process.env.HARAVAN_ACCESS_TOKEN;

    // If using OAuth, try to load stored token
    if (options.oauth || options.appId) {
      const appId = options.appId || process.env.HARAVAN_APP_ID;
      if (appId) {
        const stored = loadToken(appId);
        if (stored) {
          if (isTokenExpired(stored)) {
            logger.error(
              'Stored token is expired. Run `haravan-mcp login` to refresh.'
            );
            process.exit(1);
          }
          accessToken = stored.accessToken;
          logger.info(`Using stored OAuth token for app: ${appId}`);
        }
      }
    }

    if (!accessToken) {
      logger.error(
        'No access token provided. Use -t <token>, set HARAVAN_ACCESS_TOKEN env var, or run `haravan-mcp login` first.'
      );
      process.exit(1);
    }

    // Resolve tool filter
    const toolFilter = options.tools
      ? resolveToolFilter(options.tools)
      : undefined;

    const config = mergeConfig({
      accessToken,
      appId: options.appId,
      appSecret: options.appSecret,
      domain: options.domain,
      webhookDomain: options.webhookDomain,
      tools: toolFilter,
      mode: options.mode === 'http' ? 'http' : 'stdio',
      port: parseInt(options.port),
      debug: options.debug || false,
    });

    const { mcpServer, toolCount, createServer } = initHaravanMcpServer(config);

    logger.info(`Haravan MCP Server v${VERSION}`);
    logger.info(`Tools enabled: ${toolCount}/${allTools.length}`);
    logger.info(`Transport: ${config.mode}`);

    if (config.mode === 'http') {
      await startHttpTransport(createServer, config.port);
    } else {
      await startStdioTransport(mcpServer);
    }
  });

// ============================================
// Login Command - OAuth 2.0 Flow
// ============================================
program
  .command('login')
  .description('Login to Haravan via OAuth 2.0')
  .requiredOption('-a, --app-id <appId>', 'Haravan App ID')
  .requiredOption('-s, --app-secret <appSecret>', 'Haravan App Secret')
  .option(
    '--scope <scope>',
    'OAuth scopes (space or comma separated)',
    'openid profile email org userinfo grant_service wh_api com.write_products com.write_orders com.write_customers com.write_inventories com.read_shop web.write_contents'
  )
  .option('-p, --port <port>', 'Local callback port', '3000')
  .option(
    '--redirect-url <url>',
    'Custom redirect URL (default: http://localhost:PORT/callback)'
  )
  .option('--debug', 'Enable debug logging')
  .action(async (options) => {
    if (options.debug) {
      setLogLevel(LogLevel.DEBUG);
    }

    const scopes = options.scope
      .split(/[,\s]+/)
      .filter((s: string) => s.trim());

    logger.info(`Starting OAuth login for app: ${options.appId}`);
    logger.info(`Scopes: ${scopes.join(', ')}`);

    try {
      await performOAuthLogin({
        appId: options.appId,
        appSecret: options.appSecret,
        scope: scopes,
        port: parseInt(options.port),
        redirectUrl: options.redirectUrl,
      });

      logger.info('Login successful! Token stored.');
      logger.info(
        `You can now start the MCP server: haravan-mcp mcp -a ${options.appId} --oauth`
      );
    } catch (err: any) {
      logger.error('Login failed:', err.message);
      process.exit(1);
    }
  });

// ============================================
// Logout Command
// ============================================
program
  .command('logout')
  .description('Remove stored OAuth tokens')
  .option('-a, --app-id <appId>', 'Remove token for specific app')
  .option('--all', 'Remove all stored tokens')
  .action((options) => {
    if (options.all) {
      const tokens = loadAllTokens();
      for (const appId of Object.keys(tokens)) {
        removeToken(appId);
      }
      logger.info('All tokens removed.');
    } else if (options.appId) {
      removeToken(options.appId);
      logger.info(`Token removed for app: ${options.appId}`);
    } else {
      logger.error('Specify --app-id or --all');
      process.exit(1);
    }
  });

// ============================================
// Whoami Command - Show stored sessions
// ============================================
program
  .command('whoami')
  .description('Show stored token information')
  .action(() => {
    const tokens = loadAllTokens();
    const appIds = Object.keys(tokens);

    if (appIds.length === 0) {
      logger.info('No stored tokens found. Run `haravan-mcp login` first.');
      return;
    }

    for (const appId of appIds) {
      const token = tokens[appId];
      const expired = isTokenExpired(token);
      const masked =
        token.accessToken && token.accessToken.length >= 12
          ? `${token.accessToken.substring(0, 8)}...${token.accessToken.substring(token.accessToken.length - 4)}`
          : '***';

      console.log(`\nApp ID: ${appId}`);
      console.log(`  Token: ${masked}`);
      console.log(`  Status: ${expired ? 'Expired' : 'Active'}`);
      console.log(
        `  Created: ${new Date(token.createdAt).toISOString()}`
      );
      if (token.expiresAt) {
        console.log(
          `  Expires: ${new Date(token.expiresAt).toISOString()}`
        );
      }
      if (token.scope) {
        console.log(`  Scopes: ${token.scope.join(', ')}`);
      }
    }
  });

// ============================================
// Tools Command - List available tools
// ============================================
program
  .command('tools')
  .description('List all available tools and presets')
  .option('--presets', 'Show only presets')
  .option('--project <project>', 'Filter by project (customers, orders, etc.)')
  .action((options) => {
    if (options.presets) {
      console.log('\nAvailable Presets:');
      console.log('=================');
      for (const [name, tools] of Object.entries(PRESETS)) {
        console.log(`\n  ${name} (${tools.length} tools):`);
        for (const t of tools) {
          console.log(`    - ${t}`);
        }
      }
      return;
    }

    let tools = allTools;
    if (options.project) {
      tools = tools.filter((t) => t.project === options.project);
    }

    console.log(`\nAvailable Tools (${tools.length} total):`);
    console.log('=====================================');

    const grouped = tools.reduce(
      (acc, t) => {
        if (!acc[t.project]) acc[t.project] = [];
        acc[t.project].push(t);
        return acc;
      },
      {} as Record<string, typeof tools>
    );

    for (const [project, projectTools] of Object.entries(grouped)) {
      console.log(`\n  [${project}] (${projectTools.length} tools)`);
      for (const t of projectTools) {
        const desc = t.description.length > 80
          ? t.description.substring(0, 80) + '...'
          : t.description;
        console.log(`    ${t.name}`);
        console.log(`      ${desc}`);
      }
    }
  });

program.parse(process.argv);
