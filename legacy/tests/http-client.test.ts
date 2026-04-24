import { HaravanClient } from '../src/utils/http-client';

describe('HaravanClient', () => {
  test('should create client with options', () => {
    const client = new HaravanClient({
      domain: 'https://apis.haravan.com',
      accessToken: 'test-token',
    });
    expect(client).toBeTruthy();
  });

  test('should allow changing access token', () => {
    const client = new HaravanClient({
      domain: 'https://apis.haravan.com',
      accessToken: 'initial-token',
    });
    client.setAccessToken('new-token');
    expect(client).toBeTruthy();
  });
});
