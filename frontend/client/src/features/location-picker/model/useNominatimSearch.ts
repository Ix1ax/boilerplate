import { useQuery } from '@tanstack/react-query';

import { useDebounce } from '@/shared/hooks';

export type PlaceResult = {
  display_name: string;
  lat: string;
  lon: string;
};

// Смещение поиска к Красноярску (демо-город), без жёсткого ограничения региона.
const VIEWBOX = '92.5,56.2,93.3,55.8';

/**
 * Поиск адреса через Nominatim (OpenStreetMap) — бесплатно, без API-ключа.
 * Запросы редкие (debounce 400мс), в соответствии с политикой использования сервиса.
 */
export const useNominatimSearch = (query: string) => {
  const debounced = useDebounce(query, 400);
  const trimmed = debounced.trim();

  return useQuery({
    queryKey: ['nominatim', 'search', trimmed],
    queryFn: async (): Promise<PlaceResult[]> => {
      const params = new URLSearchParams({
        format: 'jsonv2',
        q: trimmed,
        limit: '5',
        viewbox: VIEWBOX,
        'accept-language': 'ru',
      });
      const response = await fetch(
        `https://nominatim.openstreetmap.org/search?${params}`,
      );
      if (!response.ok) throw new Error('Поиск адреса не удался');
      return response.json();
    },
    enabled: trimmed.length >= 3,
    staleTime: 60_000,
    meta: { silent: true },
  });
};
