import { mergeConfig, validateDomain, DEFAULT_CONFIG } from '../src/utils/config';

describe('Config', () => {
  test('should merge with defaults', () => {
    const config = mergeConfig({ accessToken: 'test' });
    expect(config.accessToken).toBe('test');
    expect(config.domain).toBe('https://apis.haravan.com');
    expect(config.mode).toBe('stdio');
  });

  test('should override defaults', () => {
    const config = mergeConfig({ mode: 'http', port: 8080 });
    expect(config.mode).toBe('http');
    expect(config.port).toBe(8080);
  });
});

describe('Domain Validation', () => {
  test('should accept valid Haravan domains', () => {
    expect(validateDomain('https://apis.haravan.com')).toBe(true);
    expect(validateDomain('https://webhook.haravan.com')).toBe(true);
    expect(validateDomain('https://accounts.haravan.com')).toBe(true);
    expect(validateDomain('https://my-store.haravan.com')).toBe(true);
  });

  test('should reject non-Haravan domains', () => {
    expect(validateDomain('https://evil.com')).toBe(false);
    expect(validateDomain('https://haravan.com.evil.com')).toBe(false);
    expect(validateDomain('http://apis.haravan.com')).toBe(false); // HTTP not HTTPS
    expect(validateDomain('https://apis.haravan.com/path')).toBe(false); // Has path
  });

  test('mergeConfig should throw on invalid domain', () => {
    expect(() =>
      mergeConfig({ domain: 'https://evil.com' })
    ).toThrow('Invalid API domain');
  });

  test('mergeConfig should throw on invalid webhook domain', () => {
    expect(() =>
      mergeConfig({ webhookDomain: 'http://localhost:9999' })
    ).toThrow('Invalid webhook domain');
  });

  test('mergeConfig should accept valid custom domain', () => {
    const config = mergeConfig({ domain: 'https://apis.haravan.com' });
    expect(config.domain).toBe('https://apis.haravan.com');
  });
});
