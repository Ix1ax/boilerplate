import { CloudSunIcon } from 'lucide-react';

import { useEventWeather } from '@/entities/event';

/** Прогноз для уличной встречи: температура, осадки и главный совет. */
export const EventWeatherNote = ({ eventId }: { eventId: string }) => {
  const { data } = useEventWeather(eventId, true);
  if (!data) return null;

  if (!data.available || !data.forecast) {
    return data.reason ? (
      <p className="text-[15px] text-muted-foreground">Погода: {data.reason}</p>
    ) : null;
  }

  const { temperature_c, description, precipitation_probability, advice } =
    data.forecast;
  return (
    <div className="flex items-center gap-4 rounded-[22px] bg-card px-5 py-4">
      <CloudSunIcon
        aria-hidden
        className="size-9 shrink-0 text-blue dark:text-interest"
      />
      <div className="flex flex-col">
        <b className="text-lg">
          {Math.round(temperature_c)}°, {description}
          {precipitation_probability > 0 &&
            ` · осадки ${precipitation_probability}%`}
        </b>
        {advice && <span>Совет: {advice}</span>}
      </div>
    </div>
  );
};
