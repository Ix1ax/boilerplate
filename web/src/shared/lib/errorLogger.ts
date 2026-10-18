import { logger } from './logger';

export type ErrorLoggerOptions = {
  level?: 'warn' | 'error';
  scope?: string;
  context?: Record<string, unknown>;
  cause?: unknown;
  silent?: boolean;
};

const describe = (cause: unknown) => {
  return cause instanceof Error ? `${cause.name}: ${cause.message}` : cause;
};

export class ErrorLogger extends Error {
  readonly scope?: string;
  readonly context?: Record<string, unknown>;

  constructor(message: string, options: ErrorLoggerOptions = {}) {
    super(message, { cause: options.cause });
    this.name = 'ErrorLogger';
    this.scope = options.scope;
    this.context = options.context;

    if (options.silent) return;

    const log = options.scope ? logger.child(options.scope) : logger;
    log[options.level ?? 'error'](message, this, {
      ...options.context,
      ...(options.cause !== undefined && { cause: describe(options.cause) }),
    });
  }

  static from = (
    error: unknown,
    message: string,
    options: ErrorLoggerOptions = {},
  ) => {
    return error instanceof ErrorLogger
      ? error
      : new ErrorLogger(message, { ...options, cause: error });
  };
}
