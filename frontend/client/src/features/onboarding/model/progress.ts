import { readLocalStorage, writeLocalStorage } from '@/shared/lib';

import type { Tour, TourStatus } from './types';

const key = (tour: Tour, userId: string) =>
  `onboarding:${encodeURIComponent(userId)}:${encodeURIComponent(tour.id)}:v${tour.version}`;

export const getTourStatus = (
  tour: Tour,
  userId: string,
): TourStatus | null => {
  const value = readLocalStorage(key(tour, userId));
  return value === 'started' || value === 'completed' || value === 'dismissed'
    ? value
    : null;
};

export const saveTourStatus = (
  tour: Tour,
  userId: string,
  status: TourStatus,
) => {
  writeLocalStorage(key(tour, userId), status);
};
