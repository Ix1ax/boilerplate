import { timingSafeEqual } from 'node:crypto';
import { cookies } from 'next/headers';
import { NextResponse } from 'next/server';
import {
  apiTarget,
  appOrigin,
  cookieOptions,
  exchange,
  saveTokens,
} from '@/shared/auth/server';
export const runtime = 'nodejs';
export const GET = async (request: Request) => {
  const jar = await cookies();
  const saved = jar.get('starter_oidc')?.value;
  jar.set('starter_oidc', '', cookieOptions('/auth', 0));
  const params = new URL(request.url).searchParams;
  const code = params.get('code');
  const state = params.get('state');
  try {
    const previous = JSON.parse(saved ?? '{}') as {
      state?: string;
      verifier?: string;
    };
    if (
      !code ||
      !state ||
      !previous.state ||
      !previous.verifier ||
      Buffer.byteLength(state) !== Buffer.byteLength(previous.state) ||
      !timingSafeEqual(Buffer.from(state), Buffer.from(previous.state))
    )
      throw new Error('Не совпадает состояние запроса авторизации');
    const tokens = await exchange({
      grant_type: 'authorization_code',
      code,
      redirect_uri: `${appOrigin()}/auth/callback`,
      code_verifier: previous.verifier,
    });
    if (!tokens) throw new Error('Не удалось обменять код авторизации');
    // Сохраняем сессию только после проверки подписи и параметров токена в API.
    const identity = await fetch(`${apiTarget()}/api/v1/me`, {
      headers: { Authorization: `Bearer ${tokens.access_token}` },
      cache: 'no-store',
      signal: AbortSignal.timeout(10_000),
    });
    if (!identity.ok) throw new Error('Токен доступа не прошёл проверку');
    await saveTokens(tokens);
    return NextResponse.redirect(`${appOrigin()}/`);
  } catch {
    return NextResponse.redirect(`${appOrigin()}/?auth_error=1`);
  }
};
