import { type Event, LEVEL_FIRST_MEETING } from '@/entities/event';

import { Button } from '@/shared/ui';

type Row = { label: string; value: string | null; hint?: string };

const rowsOf = (event: Event): Row[] => {
  const buddy = event.my_participation?.buddy;
  let buddyText: string | null = null;
  if (buddy) buddyText = `${buddy.name} — встретит вас на месте`;
  else if (event.buddies_count > 0)
    buddyText = `Готовы встретить новичков: ${event.buddies_count}. Отметьте «Иду впервые» при записи`;
  let meetingPoint: string | null = null;
  if (event.private_info.trim()) meetingPoint = event.private_info;
  else if (event.has_private_info)
    meetingPoint = 'Откроется после подтверждения записи';
  return [
    { label: 'Опыт', value: LEVEL_FIRST_MEETING[event.level] },
    { label: 'Что взять с собой', value: event.requirements.trim() || null },
    { label: 'Стоимость', value: event.price_note.trim() || null },
    {
      label: 'Кто встретит',
      value: buddyText,
    },
    {
      label: 'Точка сбора',
      value: meetingPoint,
    },
  ];
};

interface FirstMeetingProps {
  event: Event;
  /** Можно ли спросить: вошёл, не организатор и обсуждение открыто. */
  canAsk: boolean;
  onAsk: (topic: string) => void;
}

/** «Первая встреча»: что известно заранее. Пустой пункт — повод спросить. */
export const FirstMeeting = ({ event, canAsk, onAsk }: FirstMeetingProps) => (
  <section className="flex flex-col gap-3">
    <div>
      <h2 className="title-display text-[26px]">Первая встреча</h2>
      <p className="mt-1 text-ink-soft dark:text-muted-foreground">
        Что известно заранее. Пустой пункт — повод спросить.
      </p>
    </div>
    <dl className="rounded-3xl bg-card px-[22px] py-1">
      {rowsOf(event).map((row) => (
        <div
          key={row.label}
          className="grid min-h-[52px] grid-cols-1 items-center gap-x-4 gap-y-1 border-b border-stone py-3.5 last:border-b-0 sm:grid-cols-[220px_minmax(0,1fr)_auto] dark:border-border"
        >
          <dt className="font-bold">{row.label}</dt>
          {row.value ? (
            <dd className="whitespace-pre-line">{row.value}</dd>
          ) : (
            <dd>
              <i className="text-muted-foreground">
                Организатор пока не уточнил
              </i>
            </dd>
          )}
          {!row.value && canAsk ? (
            <Button
              variant="outline"
              size="sm"
              className="justify-self-start"
              onClick={() => onAsk(row.label)}
            >
              Спросить
            </Button>
          ) : (
            <span className="max-sm:hidden" />
          )}
        </div>
      ))}
    </dl>
  </section>
);
