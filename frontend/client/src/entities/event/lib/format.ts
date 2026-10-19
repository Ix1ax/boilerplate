import type { Event, EventLevel, JoinMode, Organizer } from '../model/types';

// Все времена показываем по Красноярску — городу демо, а не по часовому поясу браузера.
export const CITY_TIME_ZONE = 'Asia/Krasnoyarsk';

const fmt = (options: Intl.DateTimeFormatOptions) =>
  new Intl.DateTimeFormat('ru-RU', { timeZone: CITY_TIME_ZONE, ...options });

const weekdayShort = fmt({ weekday: 'short' });
const weekdayLong = fmt({ weekday: 'long' });
const dayMonth = fmt({ day: 'numeric', month: 'long' });
const dayMonthNum = fmt({ day: '2-digit', month: '2-digit' });
const time = fmt({ hour: '2-digit', minute: '2-digit' });
const isoDate = new Intl.DateTimeFormat('en-CA', {
  timeZone: CITY_TIME_ZONE,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
});

const capitalize = (value: string) =>
  value.charAt(0).toUpperCase() + value.slice(1);

/** Дата в городе демо: `2026-10-20`. */
export const toCityDate = (value: Date | string) =>
  isoDate.format(new Date(value));

/** «12:00» */
export const formatTime = (value: string) => time.format(new Date(value));

/** «вт 20.10 · 12:00» — для меток на обложках. */
export const formatWhen = (value: string) =>
  `${weekdayShort.format(new Date(value))} ${dayMonthNum.format(new Date(value))} · ${formatTime(value)}`;

/** «Вторник, 20 октября» */
export const formatDayLong = (value: string) =>
  `${capitalize(weekdayLong.format(new Date(value)))}, ${dayMonth.format(new Date(value))}`;

/** «Вт, 20 октября» */
export const formatDayShort = (value: string) =>
  `${capitalize(weekdayShort.format(new Date(value)))}, ${dayMonth.format(new Date(value))}`;

/** «Завтра, вт 20 октября» / «Сегодня, …» / «Ср, 21 октября». */
export const formatDayRelative = (value: string, now = new Date()) => {
  const day = toCityDate(value);
  const today = toCityDate(now);
  const tomorrow = toCityDate(new Date(now.getTime() + 86_400_000));
  const short = `${weekdayShort.format(new Date(value))} ${dayMonth.format(new Date(value))}`;
  if (day === today) return `Сегодня, ${short}`;
  if (day === tomorrow) return `Завтра, ${short}`;
  return formatDayShort(value);
};

export const LEVEL_LABELS: Record<EventLevel, string> = {
  any: 'Любой уровень',
  beginner: 'Без опыта',
  amateur: 'Есть опыт',
  pro: 'Опытные',
};

/** Подпись уровня строчными — «для кого» в карточке. */
export const levelShort = (level: EventLevel) =>
  LEVEL_LABELS[level].toLowerCase();

export const LEVEL_FIRST_MEETING: Record<EventLevel, string> = {
  any: 'Любой уровень — подойдёт и новичку',
  beginner: 'Не нужен — начнём с основ',
  amateur: 'Нужен небольшой опыт',
  pro: 'Для опытных участников',
};

export const JOIN_MODE_LABELS: Record<JoinMode, string> = {
  open: 'Свободная запись',
  request: 'По заявке',
};

export const organizerKind = (organizer: Organizer) =>
  organizer.type === 'organization' ? 'клуб' : 'человек';

/** «В составе: 4» / «В составе: 4 из 10» / «Пока никто не записался». */
export const formatTeamCount = (
  event: Pick<Event, 'confirmed_count' | 'capacity'>,
) => {
  if (event.capacity)
    return `В составе: ${event.confirmed_count} из ${event.capacity}`;
  if (event.confirmed_count === 0) return 'Пока никто не записался';
  return `В составе: ${event.confirmed_count}`;
};

export const isPast = (event: Pick<Event, 'starts_at' | 'ends_at'>) =>
  new Date(event.ends_at ?? event.starts_at).getTime() < Date.now();

export type DayRange = 'week' | 'today' | 'tomorrow' | 'weekend' | 'all';

export const DAY_RANGE_LABELS: Record<DayRange, string> = {
  week: 'Ближайшие 7 дней',
  today: 'Сегодня',
  tomorrow: 'Завтра',
  weekend: 'Выходные',
  all: 'Любая дата',
};

const addDays = (cityDate: string, days: number) => {
  const date = new Date(`${cityDate}T00:00:00Z`);
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
};

/** Границы выдачи по датам в городе демо: `to` не включается. */
export const dayRangeParams = (
  range: DayRange,
  now = new Date(),
): { from?: string; to?: string } => {
  const today = toCityDate(now);
  switch (range) {
    case 'today':
      return { from: today, to: addDays(today, 1) };
    case 'tomorrow':
      return { from: addDays(today, 1), to: addDays(today, 2) };
    case 'week':
      return { from: today, to: addDays(today, 7) };
    case 'weekend': {
      // 0 — воскресенье: выходные уже идут, ищем до понедельника.
      const weekday = new Date(`${today}T00:00:00Z`).getUTCDay();
      if (weekday === 0) return { from: today, to: addDays(today, 1) };
      const saturday = addDays(today, 6 - weekday);
      return {
        from: weekday === 6 ? today : saturday,
        to: addDays(saturday, 2),
      };
    }
    default:
      return {};
  }
};
