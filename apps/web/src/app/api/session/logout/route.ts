import { NextResponse } from 'next/server';
import { appOrigin, clearTokens, clientId, issuer } from '@/shared/auth/server';
import { isSameOrigin } from '@/shared/auth/policy';
export const POST = async (request: Request) => {
  if (!isSameOrigin(request, appOrigin()))
    return NextResponse.json(
      { error: { code: 'csrf', message: 'Недопустимый источник запроса' } },
      { status: 403 },
    );
  await clearTokens();
  const url = new URL(`${issuer()}/protocol/openid-connect/logout`);
  url.search = new URLSearchParams({
    client_id: clientId(),
    post_logout_redirect_uri: `${appOrigin()}/`,
  }).toString();
  return NextResponse.json(
    { url: url.toString() },
    { headers: { 'Cache-Control': 'no-store' } },
  );
};
