import { useState } from 'react';

const MAX_OFFSET = 100_000;

/** Постраничная навигация offset/limit (та же верхняя граница, что и у API). */
export const useOffsetPagination = (pageSize = 20) => {
  const [offset, setOffset] = useState(0);

  return {
    offset,
    pageSize,
    hasPrev: offset > 0,
    hasNext: (total: number) =>
      offset + pageSize < total && offset + pageSize <= MAX_OFFSET,
    toPrev: () => setOffset((prev) => Math.max(0, prev - pageSize)),
    toNext: () => setOffset((prev) => prev + pageSize),
  };
};
