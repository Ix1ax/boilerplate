import { AxiosError, AxiosHeaders } from 'axios';
import { expect, it } from 'vitest';

import { toApiError } from './errors';

it('показывает сообщение из вложенной ошибки Go API', () => {
  const error = new AxiosError(
    'Request failed',
    'ERR_BAD_REQUEST',
    undefined,
    undefined,
    {
      status: 400,
      statusText: 'Bad Request',
      headers: {},
      config: { headers: new AxiosHeaders() },
      data: {
        error: { code: 'validation_error', message: 'Введите название' },
      },
    },
  );
  const result = toApiError(error);
  expect(result.message).toBe('Введите название');
  expect(result.status).toBe(400);
});
