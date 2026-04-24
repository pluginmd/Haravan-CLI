import { allTools, filterTools, PRESETS, getProjects } from '../src/mcp-tool/registry';

describe('Tool Registry', () => {
  test('should have all tools defined', () => {
    expect(allTools.length).toBeGreaterThan(40);
  });

  test('all tools should have required properties', () => {
    for (const tool of allTools) {
      expect(tool.name).toBeTruthy();
      expect(tool.project).toBeTruthy();
      expect(tool.description).toBeTruthy();
      expect(tool.schema).toBeTruthy();
      expect(tool.httpMethod).toBeTruthy();
      expect(tool.path).toBeTruthy();
      expect(tool.scopes).toBeTruthy();
      expect(tool.scopes.length).toBeGreaterThan(0);
    }
  });

  test('all tool names should be unique', () => {
    const names = allTools.map((t) => t.name);
    const uniqueNames = new Set(names);
    expect(uniqueNames.size).toBe(names.length);
  });

  test('tool names should follow haravan_* or hrv_* convention', () => {
    for (const tool of allTools) {
      expect(tool.name).toMatch(/^(haravan_|hrv_)/);
    }
  });

  test('should have expected projects', () => {
    const projects = getProjects();
    expect(projects).toContain('smart');
    expect(projects).toContain('customers');
    expect(projects).toContain('orders');
    expect(projects).toContain('products');
    expect(projects).toContain('inventory');
    expect(projects).toContain('shop');
    expect(projects).toContain('webhooks');
    expect(projects).toContain('content');
  });

  test('should have 15 smart aggregation tools', () => {
    const smart = allTools.filter((t) => t.project === 'smart');
    expect(smart.length).toBe(15);
    for (const tool of smart) {
      expect(tool.customHandler).toBeTruthy();
    }
  });

  test('filterTools should return all tools when no filter', () => {
    const result = filterTools(allTools, undefined);
    expect(result.length).toBe(allTools.length);
  });

  test('filterTools should filter by project name', () => {
    const result = filterTools(allTools, ['customers']);
    expect(result.length).toBeGreaterThan(0);
    for (const tool of result) {
      expect(tool.project).toBe('customers');
    }
  });

  test('filterTools should filter by exact tool name', () => {
    const result = filterTools(allTools, ['haravan_shop_get']);
    expect(result.length).toBe(1);
    expect(result[0].name).toBe('haravan_shop_get');
  });

  test('filterTools should handle "all"', () => {
    const result = filterTools(allTools, ['all']);
    expect(result.length).toBe(allTools.length);
  });

  test('filterTools should expand presets', () => {
    const result = filterTools(allTools, ['preset.default']);
    expect(result.length).toBeGreaterThan(0);
    expect(result.length).toBeLessThan(allTools.length);
  });
});

describe('Presets', () => {
  test('all presets should reference valid tool names', () => {
    const toolNames = new Set(allTools.map((t) => t.name));
    for (const [presetName, tools] of Object.entries(PRESETS)) {
      for (const toolName of tools) {
        expect(toolNames.has(toolName)).toBe(true);
      }
    }
  });

  test('preset.default should have read-only tools', () => {
    const defaultTools = PRESETS['preset.default'];
    expect(defaultTools).toBeTruthy();
    expect(defaultTools.length).toBeGreaterThan(5);
  });

  test('preset.light should be smaller than preset.default', () => {
    expect(PRESETS['preset.light'].length).toBeLessThan(
      PRESETS['preset.default'].length
    );
  });
});
