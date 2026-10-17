import { type AxiosAdapter, AxiosError } from 'axios';
import { afterEach, expect, it, vi } from 'vitest';

import { API, request } from './client';
import { configureApi } from './config';

const originalAdapter = API.defaults.adapter;

afterEach(() => {
  API.defaults.adapter = originalAdapter;
  configureApi({ onUnauthorized: () => {} });
});

const rejectWith =
  (status: number): AxiosAdapter =>
  (config) =>
    Promise.reject(
      new AxiosError('Ошибка запроса', undefined, config, undefined, {
        status,
        statusText: 'Ошибка',
        headers: {},
        config,
        data: { error: { message: 'Ошибка сервера' } },
      }),
    );

it('401 на /me не вызывает редирект и дополнительные запросы', async () => {
  const onUnauthorized = vi.fn();
  const adapter = vi.fn(rejectWith(401));
  configureApi({ onUnauthorized });
  API.defaults.adapter = adapter;

  await expect(request({ url: '/me' })).rejects.toMatchObject({ status: 401 });
  expect(onUnauthorized).not.toHaveBeenCalled();
  expect(adapter).toHaveBeenCalledTimes(1);
});

it('401 защищённого запроса вызывает переход на вход без повтора операции', async () => {
  const onUnauthorized = vi.fn();
  const adapter = vi.fn(rejectWith(401));
  configureApi({ onUnauthorized });
  API.defaults.adapter = adapter;

  await expect(
    request({ url: '/tasks', method: 'POST', data: { title: 'Задача' } }),
  ).rejects.toMatchObject({ status: 401 });
  expect(onUnauthorized).toHaveBeenCalledTimes(1);
  expect(adapter).toHaveBeenCalledTimes(1);
});

it.each(['/me', '/tasks'])(
  'сбой сервера на %s сохраняет авторизацию',
  async (url) => {
    const onUnauthorized = vi.fn();
    configureApi({ onUnauthorized });
    API.defaults.adapter = rejectWith(503);

    await expect(request({ url })).rejects.toMatchObject({ status: 503 });
    expect(onUnauthorized).not.toHaveBeenCalled();
  },
);

it('сетевая ошибка не вызывает переход на вход', async () => {
  const onUnauthorized = vi.fn();
  configureApi({ onUnauthorized });
  API.defaults.adapter = (config) =>
    Promise.reject(new AxiosError('Нет соединения', 'ERR_NETWORK', config));

  await expect(request({ url: '/me' })).rejects.toMatchObject({ status: null });
  expect(onUnauthorized).not.toHaveBeenCalled();
});
