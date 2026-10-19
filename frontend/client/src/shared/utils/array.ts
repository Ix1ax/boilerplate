/**
 * Группирует элементы по ключу. Аккумулятор — объект без прототипа
 * (`Object.create(null)`), чтобы ключи `constructor`/`__proto__` не ловили
 * унаследованные значения. Тип — `Partial`, т.к. присутствуют только
 * реально встреченные группы.
 */
export const groupBy = <T, K extends PropertyKey>(
  items: readonly T[],
  getKey: (item: T) => K,
): Partial<Record<K, T[]>> => {
  const result = Object.create(null) as Record<K, T[]>;
  for (const item of items) {
    const key = getKey(item);
    (result[key] ??= []).push(item);
  }
  return result;
};

/** Оставляет по одному элементу на каждый ключ, сохраняя порядок. */
export const uniqueBy = <T, K>(
  items: readonly T[],
  getKey: (item: T) => K,
): T[] => {
  const seen = new Set<K>();
  const result: T[] = [];
  for (const item of items) {
    const key = getKey(item);
    if (seen.has(key)) continue;
    seen.add(key);
    result.push(item);
  }
  return result;
};
