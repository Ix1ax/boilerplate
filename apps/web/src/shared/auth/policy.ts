export const isSameOrigin = (request: Request, origin: string): boolean =>
  request.headers.get('origin') === new URL(origin).origin;

export const isAllowedApiPath = (segments: string[]): boolean =>
  segments.length > 0 &&
  segments[0] === 'v1' &&
  segments.every((segment) => /^[a-zA-Z0-9_-]+$/.test(segment));
