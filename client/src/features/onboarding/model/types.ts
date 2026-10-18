export type TourStatus = 'started' | 'completed' | 'dismissed';

export type TourStep = {
  /** Значение data-tour; отсутствие означает вводный шаг без подсветки. */
  target?: string;
  title: string;
  description: string;
};

export type Tour = {
  id: string;
  version: number;
  steps: readonly TourStep[];
};
