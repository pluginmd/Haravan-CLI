import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import { logger } from '../utils/logger';

export interface StoredToken {
  accessToken: string;
  refreshToken?: string;
  expiresAt?: number;
  appId?: string;
  scope?: string[];
  createdAt: number;
}

const TOKEN_DIR = path.join(os.homedir(), '.haravan-mcp');
const TOKEN_FILE = path.join(TOKEN_DIR, 'tokens.json');

function ensureDir() {
  if (!fs.existsSync(TOKEN_DIR)) {
    fs.mkdirSync(TOKEN_DIR, { recursive: true, mode: 0o700 });
  }
}

/**
 * Atomic write: write to .tmp file, then rename.
 * Prevents corruption if process is killed mid-write.
 */
function atomicWriteJson(filePath: string, data: any): void {
  const tmpPath = filePath + '.tmp';
  fs.writeFileSync(tmpPath, JSON.stringify(data, null, 2), { mode: 0o600 });
  fs.renameSync(tmpPath, filePath);
}

/**
 * Store an access token for later use.
 */
export function storeToken(appId: string, token: StoredToken): void {
  ensureDir();
  const tokens = loadAllTokens();
  tokens[appId] = token;
  atomicWriteJson(TOKEN_FILE, tokens);
  logger.info(`Token stored for app: ${appId}`);
}

/**
 * Load a stored token by app ID.
 */
export function loadToken(appId: string): StoredToken | null {
  const tokens = loadAllTokens();
  return tokens[appId] || null;
}

/**
 * Load all stored tokens.
 */
export function loadAllTokens(): Record<string, StoredToken> {
  try {
    if (fs.existsSync(TOKEN_FILE)) {
      const data = fs.readFileSync(TOKEN_FILE, 'utf-8');
      return JSON.parse(data);
    }
  } catch (err) {
    logger.error('Failed to load tokens:', err);
  }
  return {};
}

/**
 * Remove a stored token.
 */
export function removeToken(appId: string): void {
  const tokens = loadAllTokens();
  delete tokens[appId];
  ensureDir();
  atomicWriteJson(TOKEN_FILE, tokens);
  logger.info(`Token removed for app: ${appId}`);
}

/**
 * Check if a token is expired.
 */
export function isTokenExpired(token: StoredToken): boolean {
  if (!token.expiresAt) return false;
  return Date.now() > token.expiresAt;
}
