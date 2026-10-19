const SNAKE_CASE = /_([a-z0-9])/g;

const toCamelKey = (key: string) =>
  key.replace(SNAKE_CASE, (_, char: string) => char.toUpperCase());

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

/**
 * Рекурсивно переводит ключи snake_case в camelCase. Применяется к телу каждого
 * ответа API (shared/api/client.ts) — дальше по приложению поля уже camelCase.
 * Значения (строки, числа, даты-строки) не трогаем, только имена ключей.
 */
export const deepCamelKeys = (value: unknown): unknown => {
  if (Array.isArray(value)) return value.map(deepCamelKeys);
  if (isPlainObject(value)) {
    return Object.fromEntries(
      Object.entries(value).map(([key, val]) => [
        toCamelKey(key),
        deepCamelKeys(val),
      ]),
    );
  }
  return value;
};
