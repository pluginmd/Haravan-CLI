import { resolveToolFilter, listPresets } from '../src/mcp-tool/presets';

describe('Tool Filter Resolution', () => {
  test('should return undefined for empty input', () => {
    expect(resolveToolFilter(undefined)).toBeUndefined();
    expect(resolveToolFilter('')).toBeUndefined();
  });

  test('should expand preset names', () => {
    const result = resolveToolFilter('preset.default');
    expect(result).toBeTruthy();
    expect(result!.length).toBeGreaterThan(5);
    expect(result!).toContain('haravan_shop_get');
  });

  test('should handle "all" keyword', () => {
    const result = resolveToolFilter('all');
    expect(result).toEqual(['all']);
  });

  test('should handle comma-separated values', () => {
    const result = resolveToolFilter('haravan_shop_get,haravan_products_list');
    expect(result).toEqual(['haravan_shop_get', 'haravan_products_list']);
  });

  test('should handle space-separated values', () => {
    const result = resolveToolFilter('haravan_shop_get haravan_products_list');
    expect(result).toEqual(['haravan_shop_get', 'haravan_products_list']);
  });

  test('should handle mixed presets and tool names', () => {
    const result = resolveToolFilter('preset.light,haravan_webhooks_list');
    expect(result).toBeTruthy();
    expect(result!).toContain('haravan_shop_get');
    expect(result!).toContain('haravan_webhooks_list');
  });

  test('should deduplicate results', () => {
    const result = resolveToolFilter('preset.default,haravan_shop_get');
    const occurrences = result!.filter((t) => t === 'haravan_shop_get');
    expect(occurrences.length).toBe(1);
  });

  test('should pass through project names', () => {
    const result = resolveToolFilter('products');
    expect(result).toEqual(['products']);
  });
});

describe('List Presets', () => {
  test('should return preset info', () => {
    const presets = listPresets();
    expect(presets.length).toBeGreaterThan(0);
    for (const preset of presets) {
      expect(preset.name).toMatch(/^preset\./);
      expect(preset.toolCount).toBeGreaterThan(0);
    }
  });
});
