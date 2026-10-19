import { expect, it } from 'vitest';

import { deepCamelKeys } from './caseTransform';

it('переводит ключи объекта в camelCase рекурсивно', () => {
  expect(
    deepCamelKeys({
      starts_at: '2026-10-20',
      sport: { is_custom: true, sport_id: 'abc' },
      items: [{ created_at: '1', decision_reason: 'x' }],
    }),
  ).toEqual({
    startsAt: '2026-10-20',
    sport: { isCustom: true, sportId: 'abc' },
    items: [{ createdAt: '1', decisionReason: 'x' }],
  });
});

it('не трогает значения — массивы и примитивы остаются как есть', () => {
  expect(deepCamelKeys('text')).toBe('text');
  expect(deepCamelKeys(42)).toBe(42);
  expect(deepCamelKeys(null)).toBe(null);
  expect(deepCamelKeys([1, 2, 3])).toEqual([1, 2, 3]);
});

it('ключи без подчёркивания не меняются', () => {
  expect(deepCamelKeys({ id: '1', name: 'Анна' })).toEqual({
    id: '1',
    name: 'Анна',
  });
});
