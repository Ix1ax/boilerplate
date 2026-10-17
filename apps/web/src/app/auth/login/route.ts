import { createHash, randomBytes } from 'node:crypto';
import { NextResponse } from 'next/server';
import {
  appOrigin,
  clientId,
  cookieOptions,
  issuer,
} from '@/shared/auth/server';
export const runtime = 'nodejs';
export const GET = async () => {
  const verifier = randomBytes(32).toString('base64url');
  const state = randomBytes(32).toString('base64url');
  const url = new URL(`${issuer()}/protocol/openid-connect/auth`);
  url.search = new URLSearchParams({
    client_id: clientId(),
    response_type: 'code',
    scope: 'openid profile email',
    redirect_uri: `${appOrigin()}/auth/callback`,
    state,
    code_challenge: createHash('sha256').update(verifier).digest('base64url'),
    code_challenge_method: 'S256',
  }).toString();
  const response = NextResponse.redirect(url);
  response.cookies.set(
    'starter_oidc',
    JSON.stringify({ verifier, state }),
    cookieOptions('/auth', 600),
  );
  response.headers.set('Cache-Control', 'no-store');
  return response;
};
