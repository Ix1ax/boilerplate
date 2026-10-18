import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { getTourStatus, saveTourStatus } from './progress';
import type { Tour } from './types';

const tour: Tour = { id: 'overview', version: 1, steps: [] };

describe('результат обучения', () => {
  beforeEach(() => {
    const values = new Map<string, string>();
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => values.set(key, value),
    });
  });
  afterEach(() => vi.unstubAllGlobals());

  it('сохраняет первый показ до завершения тура', () => {
    saveTourStatus(tour, 'first-visit', 'started');
    expect(getTourStatus(tour, 'first-visit')).toBe('started');
    saveTourStatus(tour, 'first-visit', 'completed');
    expect(getTourStatus(tour, 'first-visit')).toBe('completed');
  });

  it('разделяет пользователей, сценарии и версии', () => {
    saveTourStatus(tour, 'one', 'completed');
    expect(getTourStatus(tour, 'one')).toBe('completed');
    expect(getTourStatus(tour, 'two')).toBeNull();
    expect(getTourStatus({ ...tour, version: 2 }, 'one')).toBeNull();
    expect(getTourStatus({ ...tour, id: 'files' }, 'one')).toBeNull();
  });

  it('сохраняет пропуск отдельно от завершения', () => {
    saveTourStatus(tour, 'one', 'dismissed');
    expect(getTourStatus(tour, 'one')).toBe('dismissed');
  });

  it('работает при заблокированном хранилище', () => {
    vi.stubGlobal('localStorage', {
      getItem: () => {
        throw new Error('denied');
      },
      setItem: () => {
        throw new Error('denied');
      },
    });
    expect(() => saveTourStatus(tour, 'one', 'completed')).not.toThrow();
    expect(getTourStatus(tour, 'one')).toBe('completed');
  });
});
