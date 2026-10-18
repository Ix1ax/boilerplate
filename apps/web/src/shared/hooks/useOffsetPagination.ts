import { useSearchParams } from 'react-router';

const MAX_OFFSET = 100_000;

const parseOffset = (value: string | null): number => {
  const parsed = Number(value);

  return Number.isFinite(parsed) && parsed > 0 ? Math.floor(parsed) : 0;
};

/**
 * Постраничная навигация offset/limit (та же верхняя граница, что и у API).
 * Текущая страница живёт в query-параметре — переживает reload и копируется в ссылке.
 */
export const useOffsetPagination = (pageSize = 20, param = 'offset') => {
  const [searchParams, setSearchParams] = useSearchParams();
  const offset = parseOffset(searchParams.get(param));

  const setOffset = (next: number) => {
    setSearchParams((prev) => {
      const params = new URLSearchParams(prev);
      if (next > 0) params.set(param, String(next));
      else params.delete(param);
      return params;
    });
  };

  return {
    offset,
    pageSize,
    hasPrev: offset > 0,
    hasNext: (total: number) =>
      offset + pageSize < total && offset + pageSize <= MAX_OFFSET,
    toPrev: () => setOffset(Math.max(0, offset - pageSize)),
    toNext: () => setOffset(offset + pageSize),
  };
};
