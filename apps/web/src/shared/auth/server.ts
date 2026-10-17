import 'server-only';
import { cookies } from 'next/headers';

export const appOrigin = () =>
  new URL(process.env.APP_ORIGIN ?? 'http://localhost:13000').origin;
export const issuer = () =>
  process.env.OIDC_ISSUER ?? 'http://localhost:18180/realms/hackathon';
export const internalIssuer = () => process.env.OIDC_INTERNAL_URL ?? issuer();
export const clientId = () => process.env.OIDC_CLIENT_ID ?? 'hackathon-web';
export const apiTarget = () =>
  process.env.API_PROXY_TARGET ?? 'http://localhost:18080';
const secure = () =>
  process.env.SESSION_COOKIE_SECURE === 'true' ||
  (process.env.SESSION_COOKIE_SECURE !== 'false' &&
    process.env.NODE_ENV === 'production');
export const cookieOptions = (path: string, maxAge: number) => ({
  httpOnly: true,
  secure: secure(),
  sameSite: 'lax' as const,
  path,
  maxAge,
});
export type Tokens = {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  refresh_expires_in: number;
};
export const exchange = async (
  params: Record<string, string>,
): Promise<Tokens | null> => {
  const response = await fetch(
    `${internalIssuer()}/protocol/openid-connect/token`,
    {
      method: 'POST',
      body: new URLSearchParams({ client_id: clientId(), ...params }),
      cache: 'no-store',
      signal: AbortSignal.timeout(10_000),
    },
  );
  if (!response.ok) {
    if (response.status === 400 || response.status === 401) return null;
    throw new Error('Сервис авторизации временно недоступен');
  }
  const result = (await response.json()) as Partial<Tokens>;
  if (
    typeof result.access_token !== 'string' ||
    !result.access_token ||
    typeof result.refresh_token !== 'string' ||
    !result.refresh_token ||
    !Number.isSafeInteger(result.expires_in) ||
    result.expires_in! < 1 ||
    !Number.isSafeInteger(result.refresh_expires_in) ||
    result.refresh_expires_in! < 1
  )
    return null;
  return result as Tokens;
};
export const saveTokens = async (tokens: Tokens) => {
  const jar = await cookies();
  jar.set(
    'starter_access',
    tokens.access_token,
    cookieOptions('/api', tokens.expires_in),
  );
  jar.set(
    'starter_refresh',
    tokens.refresh_token,
    cookieOptions('/api', tokens.refresh_expires_in),
  );
};
export const clearTokens = async () => {
  const jar = await cookies();
  for (const name of ['starter_access', 'starter_refresh'])
    jar.set(name, '', cookieOptions('/api', 0));
};
const refreshes = new Map<string, Promise<Tokens | null>>();
export const refresh = async (): Promise<Tokens | null> => {
  const token = (await cookies()).get('starter_refresh')?.value;
  if (!token) return null;
  let pending = refreshes.get(token);
  if (!pending) {
    pending = exchange({ grant_type: 'refresh_token', refresh_token: token });
    refreshes.set(token, pending);
    void pending.finally(() => refreshes.delete(token)).catch(() => {});
  }
  return pending;
};
