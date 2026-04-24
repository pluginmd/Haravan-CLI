import crypto from 'crypto';
import axios from 'axios';
import express from 'express';
import open from 'open';
import { logger } from '../utils/logger';
import { storeToken } from './token-store';

const HARAVAN_AUTH_URL = 'https://accounts.haravan.com/connect/authorize';
const HARAVAN_TOKEN_URL = 'https://accounts.haravan.com/connect/token';

export interface OAuthConfig {
  appId: string;
  appSecret: string;
  redirectUrl?: string;
  scope: string[];
  port?: number;
}

/**
 * Perform OAuth 2.0 login flow for Haravan.
 *
 * Flow:
 * 1. Generate CSRF state token
 * 2. Open browser to Haravan authorization URL (with state)
 * 3. User logs in and approves scopes
 * 4. Haravan redirects to local callback with authorization code + state
 * 5. Verify state matches → exchange code for access_token
 * 6. Store token for future use
 */
export async function performOAuthLogin(config: OAuthConfig): Promise<string> {
  const port = config.port || 3000;
  const redirectUri = config.redirectUrl || `http://localhost:${port}/callback`;

  // Generate CSRF state token
  const state = crypto.randomBytes(32).toString('hex');

  return new Promise<string>((resolve, reject) => {
    const app = express();
    let server: any;

    app.get('/callback', async (req, res) => {
      const code = req.query.code as string;
      const returnedState = req.query.state as string;

      // Verify CSRF state
      if (returnedState !== state) {
        res.status(403).send('Invalid state parameter — possible CSRF attack.');
        reject(new Error('OAuth state mismatch'));
        return;
      }

      if (!code) {
        res.status(400).send('No authorization code received');
        reject(new Error('No authorization code received'));
        return;
      }

      try {
        // Exchange code for access token
        const tokenResponse = await axios.post(
          HARAVAN_TOKEN_URL,
          new URLSearchParams({
            grant_type: 'authorization_code',
            code,
            redirect_uri: redirectUri,
            client_id: config.appId,
            client_secret: config.appSecret,
          }).toString(),
          {
            headers: {
              'Content-Type': 'application/x-www-form-urlencoded',
            },
          }
        );

        const { access_token, refresh_token, expires_in, scope } =
          tokenResponse.data;

        // Store token
        storeToken(config.appId, {
          accessToken: access_token,
          refreshToken: refresh_token,
          expiresAt: expires_in
            ? Date.now() + expires_in * 1000
            : undefined,
          appId: config.appId,
          scope: scope ? scope.split(' ') : config.scope,
          createdAt: Date.now(),
        });

        res.send(
          '<html><body style="font-family:sans-serif;padding:40px;text-align:center;">' +
          '<h1>Haravan MCP - Login Successful!</h1>' +
          '<p>You can close this window and return to your terminal.</p>' +
          '</body></html>'
        );

        logger.info('OAuth login successful!');

        setTimeout(() => {
          server.close();
          resolve(access_token);
        }, 1000);
      } catch (err: any) {
        const errMsg = err.response?.data || err.message;
        logger.error('Token exchange failed:', errMsg);
        res.status(500).send('Token exchange failed. Check terminal for details.');
        server.close();
        reject(err);
      }
    });

    server = app.listen(port, () => {
      const scopes = config.scope.join(' ');
      const authUrl =
        `${HARAVAN_AUTH_URL}?response_type=code` +
        `&client_id=${encodeURIComponent(config.appId)}` +
        `&scope=${encodeURIComponent(scopes)}` +
        `&redirect_uri=${encodeURIComponent(redirectUri)}` +
        `&state=${encodeURIComponent(state)}` +
        `&nonce=${Date.now()}`;

      logger.info('Opening browser for Haravan login...');
      logger.info(`Auth URL: ${authUrl}`);
      logger.info(`Listening on http://localhost:${port}/callback`);

      open(authUrl).catch(() => {
        logger.warn(
          'Could not open browser automatically. Please open this URL manually:'
        );
        logger.warn(authUrl);
      });
    });

    // Timeout after 5 minutes
    setTimeout(() => {
      server.close();
      reject(new Error('OAuth login timed out after 5 minutes'));
    }, 5 * 60 * 1000);
  });
}

/**
 * Refresh an expired access token.
 */
export async function refreshAccessToken(
  appId: string,
  appSecret: string,
  refreshToken: string
): Promise<string> {
  const response = await axios.post(
    HARAVAN_TOKEN_URL,
    new URLSearchParams({
      grant_type: 'refresh_token',
      refresh_token: refreshToken,
      client_id: appId,
      client_secret: appSecret,
    }).toString(),
    {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
    }
  );

  const { access_token, refresh_token: new_refresh_token, expires_in } = response.data;

  storeToken(appId, {
    accessToken: access_token,
    refreshToken: new_refresh_token || refreshToken,
    expiresAt: expires_in ? Date.now() + expires_in * 1000 : undefined,
    appId,
    createdAt: Date.now(),
  });

  return access_token;
}
