import { PRESETS } from './registry';

/**
 * Resolve preset and tool filter strings into a flat list of tool names/projects.
 *
 * Input examples:
 *   "preset.default,haravan_webhooks_list"
 *   "products,orders"
 *   "all"
 *
 * Returns array of tool names and project names (presets expanded).
 */
export function resolveToolFilter(input?: string): string[] | undefined {
  if (!input) return undefined;

  const entries = input
    .split(/[,\s]+/)
    .map((s) => s.trim())
    .filter(Boolean);

  if (entries.length === 0) return undefined;

  const result: string[] = [];

  for (const entry of entries) {
    if (entry === 'all') {
      return ['all'];
    }

    if (entry.startsWith('preset.') && PRESETS[entry]) {
      result.push(...PRESETS[entry]);
    } else {
      result.push(entry);
    }
  }

  return [...new Set(result)];
}

/**
 * List available presets with descriptions.
 */
export function listPresets(): Array<{ name: string; toolCount: number }> {
  return Object.entries(PRESETS).map(([name, tools]) => ({
    name,
    toolCount: tools.length,
  }));
}
