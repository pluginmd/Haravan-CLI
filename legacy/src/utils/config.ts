export interface HaravanMcpConfig {
  accessToken?: string;
  appId?: string;
  appSecret?: string;
  domain: string;
  webhookDomain: string;
  tools?: string[];
  mode: 'stdio' | 'http';
  oauth: boolean;
  scope?: string[];
  port: number;
  debug: boolean;
}

export const DEFAULT_CONFIG: HaravanMcpConfig = {
  domain: 'https://apis.haravan.com',
  webhookDomain: 'https://webhook.haravan.com',
  mode: 'stdio',
  oauth: false,
  port: 3000,
  debug: false,
};

const ALLOWED_DOMAIN_PATTERNS = [
  /^https:\/\/[a-z0-9-]+\.haravan\.com$/,
  /^https:\/\/apis\.haravan\.com$/,
  /^https:\/\/webhook\.haravan\.com$/,
  /^https:\/\/accounts\.haravan\.com$/,
];

/**
 * Validate that a domain is an allowed Haravan domain.
 * Prevents SSRF via arbitrary --domain flag.
 */
export function validateDomain(domain: string): boolean {
  return ALLOWED_DOMAIN_PATTERNS.some((pattern) => pattern.test(domain));
}

export function mergeConfig(partial: Partial<HaravanMcpConfig>): HaravanMcpConfig {
  const config = { ...DEFAULT_CONFIG, ...partial };

  // Validate custom domains
  if (partial.domain && !validateDomain(config.domain)) {
    throw new Error(
      `Invalid API domain: ${config.domain}. Must be a *.haravan.com HTTPS URL.`
    );
  }
  if (partial.webhookDomain && !validateDomain(config.webhookDomain)) {
    throw new Error(
      `Invalid webhook domain: ${config.webhookDomain}. Must be a *.haravan.com HTTPS URL.`
    );
  }

  return config;
}
