export enum LogLevel {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3,
  SILENT = 4,
}

let currentLevel = LogLevel.INFO;

export function setLogLevel(level: LogLevel) {
  currentLevel = level;
}

export const logger = {
  debug: (...args: any[]) => {
    if (currentLevel <= LogLevel.DEBUG) console.error('[DEBUG]', ...args);
  },
  info: (...args: any[]) => {
    if (currentLevel <= LogLevel.INFO) console.error('[INFO]', ...args);
  },
  warn: (...args: any[]) => {
    if (currentLevel <= LogLevel.WARN) console.error('[WARN]', ...args);
  },
  error: (...args: any[]) => {
    if (currentLevel <= LogLevel.ERROR) console.error('[ERROR]', ...args);
  },
};
