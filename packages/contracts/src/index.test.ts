import { expect, it } from 'vitest';
import { createTaskSchema } from './index';
it('обрезает пробелы и отклоняет пустое или слишком длинное название', () => {
  expect(createTaskSchema.parse({ title: '  Идея  ' })).toEqual({
    title: 'Идея',
  });
  for (const title of ['', '  ', 'а'.repeat(201)])
    expect(createTaskSchema.safeParse({ title }).success).toBe(false);
  expect(createTaskSchema.safeParse({ title: 'а'.repeat(200) }).success).toBe(
    true,
  );
  expect(createTaskSchema.safeParse({ title: '🚀'.repeat(200) }).success).toBe(
    true,
  );
  expect(createTaskSchema.safeParse({ title: '🚀'.repeat(201) }).success).toBe(
    false,
  );
});
