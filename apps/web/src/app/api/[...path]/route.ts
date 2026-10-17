import { cookies } from 'next/headers';
import { NextResponse } from 'next/server';
import {
  apiTarget,
  appOrigin,
  clearTokens,
  refresh,
  saveTokens,
} from '@/shared/auth/server';
import { isAllowedApiPath, isSameOrigin } from '@/shared/auth/policy';
export const runtime = 'nodejs';
export const dynamic = 'force-dynamic';
const MAX_BODY = 11 * 1024 * 1024;
const problem = (status: number, code: string, message: string) =>
  NextResponse.json(
    { error: { code, message } },
    { status, headers: { 'Cache-Control': 'no-store' } },
  );
const proxy = async (
  request: Request,
  context: { params: Promise<{ path: string[] }> },
) => {
  const { path } = await context.params;
  if (!isAllowedApiPath(path))
    return problem(404, 'not_found', 'Маршрут не найден');
  const mutation = !['GET', 'HEAD'].includes(request.method);
  if (mutation && !isSameOrigin(request, appOrigin()))
    return problem(403, 'csrf', 'Недопустимый источник запроса');
  let body: Uint8Array | undefined;
  if (mutation && request.body) {
    const reader = request.body.getReader();
    const chunks: Uint8Array[] = [];
    let size = 0;
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.length;
      if (size > MAX_BODY) {
        await reader.cancel();
        return problem(
          413,
          'body_too_large',
          'Размер запроса не должен превышать 11 МиБ',
        );
      }
      chunks.push(value);
    }
    body = new Uint8Array(size);
    let offset = 0;
    for (const chunk of chunks) {
      body.set(chunk, offset);
      offset += chunk.length;
    }
  }
  try {
    let token = (await cookies()).get('starter_access')?.value;
    let refreshed = false;
    if (!token) {
      const tokens = await refresh();
      if (!tokens) {
        await clearTokens();
        return problem(401, 'unauthorized', 'Войдите, чтобы продолжить');
      }
      await saveTokens(tokens);
      token = tokens.access_token;
      refreshed = true;
    }
    const target = `${apiTarget()}/api/${path.join('/')}${new URL(request.url).search}`;
    const send = (access: string) =>
      fetch(target, {
        method: request.method,
        headers: {
          Authorization: `Bearer ${access}`,
          ...(request.headers.has('content-type')
            ? { 'Content-Type': request.headers.get('content-type')! }
            : {}),
        },
        body: body as BodyInit | undefined,
        redirect: 'manual',
        cache: 'no-store',
        signal: AbortSignal.timeout(30_000),
      });
    let response = await send(token);
    if (response.status === 401 && !refreshed) {
      const tokens = await refresh();
      if (!tokens) {
        await clearTokens();
        return problem(401, 'unauthorized', 'Войдите, чтобы продолжить');
      }
      await saveTokens(tokens);
      await response.body?.cancel();
      response = await send(tokens.access_token);
    }
    const headers = new Headers({
      'Cache-Control': 'no-store',
      'X-Content-Type-Options': 'nosniff',
    });
    for (const name of [
      'content-type',
      'content-disposition',
      'x-request-id',
      'retry-after',
    ]) {
      const value = response.headers.get(name);
      if (value) headers.set(name, value);
    }
    return new Response(response.body, { status: response.status, headers });
  } catch {
    return problem(
      503,
      'upstream_unavailable',
      'Сервис временно недоступен. Повторите попытку.',
    );
  }
};
export { proxy as GET, proxy as POST, proxy as PATCH, proxy as DELETE };
