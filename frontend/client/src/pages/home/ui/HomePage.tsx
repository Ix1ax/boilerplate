import { Link } from 'react-router';
import { ArrowRightIcon } from 'lucide-react';

import { EventCard, useEvents } from '@/entities/event';

import { STATIC_ROUTES } from '@/shared/constants';
import { useSmoothScroll } from '@/shared/hooks';
import { Button, Eyebrow, RadarRings } from '@/shared/ui';

const STEPS = [
  [
    '01',
    'Найдите встречу рядом',
    'Спорт, дата и режим записи. Сразу видно, кто организует и для кого встреча.',
  ],
  [
    '02',
    'Спросите до записи',
    'Нужен ли опыт, что взять, кто встретит. Ответ организатора отмечен.',
  ],
  [
    '03',
    'Запишитесь',
    'Свободная запись — сразу в составе. По заявке — решает организатор.',
  ],
  [
    '04',
    'Получите напоминание',
    'За сутки и за 2 часа до встречи придёт уведомление, в том числе в MAX.',
  ],
  [
    '05',
    'Приходите',
    'В карточке написано, кто встретит и где именно. Можно прийти одному.',
  ],
] as const;

const TOGETHER = [
  [
    'Вы оставляете запрос',
    'Спорт, район и удобное время. Например: шахматы, вечером в будни.',
    'Запрос сохранён',
  ],
  [
    'Мы ищем совпадения',
    'Когда появляется подходящая встреча, вы получаете уведомление «Нашлась компания».',
    'Это ещё не запись',
  ],
  [
    'Организатор видит спрос',
    'При создании встречи видно, сколько человек ищут такой спорт — без имён и контактов.',
    'Встреча создана',
  ],
  [
    'Вы записываетесь сами',
    'Запись — обычная: свободно или по заявке. Интерес никого не записывает автоматически.',
    '✓ Вы записаны',
  ],
] as const;

const TAG_STYLES = [
  'border-[1.5px] border-dashed border-paper/60',
  'bg-blue-soft text-blue',
  'bg-white text-ink',
  'bg-success-soft text-success',
];

export const HomePage = () => {
  useSmoothScroll();
  const { data } = useEvents({ limit: 6 });
  const events = data?.items ?? [];

  return (
    <>
      <section className="relative overflow-hidden bg-ink bg-[linear-gradient(rgb(238_240_234/5%)_1px,transparent_1px),linear-gradient(90deg,rgb(238_240_234/5%)_1px,transparent_1px)] bg-[size:80px_80px] text-paper">
        <div className="relative mx-auto flex max-w-[1360px] flex-wrap items-center gap-10 px-4 pt-16 pb-24 sm:px-8 sm:pt-20 sm:pb-32">
          <div className="relative z-[2] min-w-0 flex-[1_1_620px]">
            <Eyebrow
              className="rise flex flex-wrap gap-x-5 gap-y-2 text-paper/80"
              style={{ '--d': '60ms' } as React.CSSProperties}
            >
              <span className="inline-flex items-center gap-2">
                <span className="size-2 rounded-full bg-gold" />
                Красноярск · демо-город
              </span>
              <span>любительский спорт рядом с домом</span>
            </Eyebrow>
            <h1
              className="rise display-caps mt-5 text-[clamp(64px,11vw,172px)] leading-[.86] tracking-[-0.045em]"
              style={{ '--d': '140ms' } as React.CSSProperties}
            >
              Найди
              <br />
              <span className="text-gold">своих</span>
            </h1>
            <p
              className="rise mt-6 max-w-[640px] text-[clamp(20px,1.9vw,25px)] leading-snug"
              style={{ '--d': '260ms' } as React.CSSProperties}
            >
              Компания для шахмат, бега, зарядки и вашего собственного вида
              спорта. А если подходящей встречи ещё нет — её можно собрать.
            </p>
            <div
              className="rise mt-9 flex flex-wrap gap-3"
              style={{ '--d': '380ms' } as React.CSSProperties}
            >
              <Button
                size="xl"
                className="lift"
                render={<Link to={STATIC_ROUTES.EVENTS} />}
              >
                Найти встречу
                <ArrowRightIcon aria-hidden />
              </Button>
              <Button
                size="xl"
                variant="gold"
                className="lift"
                render={<Link to={STATIC_ROUTES.COMPANY} />}
              >
                Собрать компанию
              </Button>
            </div>
          </div>

          <div
            aria-hidden="true"
            className="rise relative aspect-square min-w-[300px] flex-[0_1_540px]"
            style={{ '--d': '300ms' } as React.CSSProperties}
          >
            <RadarRings
              tone="gold"
              sizes={[540, 480, 420, 360, 280].reverse()}
              strength={0.5}
              className="top-1/2 left-1/2"
            />
            <div className="absolute inset-[40%] grid place-items-center rounded-full bg-blue">
              <span className="relative block aspect-[1.5] w-[52%]">
                <span className="absolute top-0 left-0 aspect-square h-full rounded-full bg-white" />
                <span className="absolute top-0 right-0 aspect-square h-full rounded-full bg-gold shadow-[0_0_0_3px_var(--color-blue)]" />
              </span>
            </div>
          </div>
        </div>
      </section>

      <section
        id="path"
        className="bg-paper px-4 py-24 text-ink sm:px-8 sm:py-28"
      >
        <div className="mx-auto max-w-[1360px]">
          <div className="flex flex-wrap items-end justify-between gap-x-12 gap-y-6">
            <h2 className="display-caps max-w-[920px] text-[clamp(40px,5.6vw,84px)] leading-[.92] tracking-[-0.035em]">
              От «не&nbsp;с&nbsp;кем» до&nbsp;«я&nbsp;в&nbsp;деле»
            </h2>
            <p className="max-w-[400px] text-[19px] text-ink-soft">
              Одна логика для любого спорта: найти, понять условия, спросить,
              записаться и прийти.
            </p>
          </div>
          <ol className="mt-16 grid grid-cols-1 gap-x-10 gap-y-10 border-t-2 border-ink sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
            {STEPS.map(([n, title, text]) => (
              <li
                key={n}
                className="flex flex-col gap-4 border-b border-line-strong pt-8 pb-10"
              >
                <span className="font-display text-[56px] leading-none font-black text-blue">
                  {n}
                </span>
                <b className="text-[21px] leading-snug">{title}</b>
                <span className="text-base leading-relaxed text-ink-soft">
                  {text}
                </span>
              </li>
            ))}
          </ol>
        </div>
      </section>

      <section
        id="near"
        className="relative overflow-hidden bg-blue py-28 text-white"
      >
        <RadarRings
          tone="white"
          sizes={[260, 440, 620]}
          strength={0.3}
          className="top-[90px] right-[130px]"
        />
        <div className="relative mx-auto flex max-w-[1360px] flex-wrap items-end justify-between gap-6 px-4 sm:px-8">
          <div>
            <Eyebrow className="text-white/85">ближайшие встречи</Eyebrow>
            <h2 className="display-caps mt-3.5 text-[clamp(44px,7vw,104px)] leading-[.9] tracking-[-0.04em]">
              Рядом
              <br />и скоро
            </h2>
          </div>
          <Button
            variant="gold"
            size="xl"
            className="lift"
            render={<Link to={STATIC_ROUTES.EVENTS} />}
          >
            Все встречи
            <ArrowRightIcon aria-hidden />
          </Button>
        </div>
        {events.length > 0 && (
          <ul className="relative mt-14 flex snap-x snap-mandatory gap-5 overflow-x-auto px-4 pt-3 pb-7 sm:px-8">
            {events.map((event) => (
              <li key={event.id} className="flex w-80 shrink-0 snap-start">
                <EventCard event={event} className="w-full" />
              </li>
            ))}
          </ul>
        )}
      </section>

      <section
        id="together"
        className="relative overflow-hidden bg-ink px-4 py-28 text-paper sm:px-8"
      >
        <RadarRings
          tone="gold"
          sizes={[420, 620]}
          strength={0.3}
          className="bottom-0 left-0"
        />
        <div className="relative mx-auto max-w-[1360px]">
          <Eyebrow className="text-gold">
            собрать компанию · главное отличие
          </Eyebrow>
          <h2 className="display-caps mt-4 max-w-[1060px] text-[clamp(36px,5vw,76px)] leading-[.94] tracking-[-0.035em]">
            Нет встречи? Соберём её из&nbsp;спроса
          </h2>
          <ol className="mt-16 grid grid-cols-[repeat(auto-fit,minmax(min(260px,100%),1fr))] gap-5">
            {TOGETHER.map(([title, text, tag], i) => (
              <li
                key={title}
                className={`flex flex-col gap-4 rounded-3xl p-7 ${i === 3 ? 'bg-blue text-white' : 'border border-paper/15 bg-night'}`}
              >
                <span
                  className={`grid size-11 place-items-center rounded-full font-display font-extrabold ${i === 3 ? 'bg-white text-blue' : 'bg-gold text-ink'}`}
                >
                  {i + 1}
                </span>
                <b className="text-[21px]">{title}</b>
                <span className="text-base opacity-85">{text}</span>
                <span
                  className={`mt-auto self-start rounded-full px-3 py-1.5 text-sm font-bold ${TAG_STYLES[i]}`}
                >
                  {tag}
                </span>
              </li>
            ))}
          </ol>
          <div className="mt-7 flex flex-wrap items-center justify-between gap-5 rounded-3xl bg-gold px-7 py-6 text-ink">
            <p className="max-w-[760px] text-[19px] leading-snug font-semibold">
              Интерес — ещё не запись. Публикация встречи никого не добавляет в
              состав автоматически, а организатор не видит ваших контактов.
            </p>
            <Button
              variant="ink"
              size="lg"
              className="lift"
              render={<Link to={STATIC_ROUTES.COMPANY} />}
            >
              Собрать компанию
            </Button>
          </div>
        </div>
      </section>

      <section id="start" className="bg-paper px-4 py-28 text-ink sm:px-8">
        <div className="mx-auto max-w-[1360px]">
          <Eyebrow className="text-blue">новичкам и старшему поколению</Eyebrow>
          <h2 className="display-caps mt-3.5 max-w-[900px] text-[clamp(40px,5.2vw,80px)] leading-[.92] tracking-[-0.035em]">
            Не страшно прийти впервые
          </h2>
          <p className="mt-7 max-w-[600px] text-xl leading-relaxed text-ink-soft">
            В каждой карточке есть блок «Первая встреча». Если организатор
            чего-то не указал, мы так и пишем — и предлагаем спросить.
          </p>
          <div className="mt-7 flex max-w-[620px] flex-col gap-2.5 text-[19px] font-bold">
            <div className="rounded-2xl bg-success-soft px-[18px] py-3.5 text-success">
              ✓ Вы записаны
            </div>
            <div className="rounded-2xl bg-warning-soft px-[18px] py-3.5 text-warning">
              ◷ Ожидает решения организатора
            </div>
            <div className="rounded-2xl bg-danger-soft px-[18px] py-3.5 text-danger">
              ✕ Организатор отклонил заявку
            </div>
          </div>
        </div>
      </section>

      <section
        id="clubs"
        className="relative overflow-hidden bg-stone px-4 py-28 text-ink sm:px-8"
      >
        <RadarRings
          tone="blue"
          sizes={[440, 640]}
          strength={0.28}
          className="top-0 right-0"
        />
        <div className="relative mx-auto max-w-[1360px]">
          <Eyebrow className="text-blue">
            клубам · секциям · сообществам
          </Eyebrow>
          <h2 className="display-caps mt-3.5 max-w-[900px] text-[clamp(40px,5.2vw,80px)] leading-[.92] tracking-[-0.035em]">
            Собирайте своё сообщество
          </h2>
          <div className="mt-14 grid grid-cols-[repeat(auto-fit,minmax(min(280px,100%),1fr))] gap-5">
            {[
              [
                '01 · заявка',
                'Расскажите о клубе',
                'Название, чем занимаетесь, какие виды спорта и где. Контакт представителя не публикуется.',
              ],
              [
                '02 · проверка',
                'Статус виден всегда',
                'При отказе администратор называет причину. Заявку можно исправить и отправить снова.',
              ],
              [
                '03 · в эфире',
                'Встречи публикуются сразу',
                'После допуска — без согласования каждой встречи. Плюс спрос: сколько людей ищут ваш спорт.',
              ],
            ].map(([k, t, d], i) => (
              <div
                key={k}
                className={`flex min-h-[260px] flex-col gap-3.5 rounded-[28px] p-7 ${i === 2 ? 'bg-ink text-paper' : 'bg-card'}`}
              >
                <span
                  className={`font-mono text-sm ${i === 2 ? 'text-gold' : 'text-muted-foreground'}`}
                >
                  {k}
                </span>
                <b className="text-2xl leading-tight">{t}</b>
                <span className={i === 2 ? 'text-paper/85' : 'text-ink-soft'}>
                  {d}
                </span>
              </div>
            ))}
          </div>
          <Button
            variant="ink"
            size="xl"
            className="lift mt-10"
            render={<Link to={STATIC_ROUTES.CLUB} />}
          >
            Подать заявку клуба
          </Button>
        </div>
      </section>

      <section className="relative overflow-hidden bg-gold px-4 pt-28 pb-24 text-ink sm:px-8">
        <RadarRings
          tone="ink"
          sizes={[360, 560]}
          strength={0.25}
          className="right-[100px] bottom-[-80px]"
        />
        <div className="relative mx-auto max-w-[1360px]">
          <p className="display-caps text-[clamp(56px,11vw,176px)] leading-[.86] tracking-[-0.045em]">
            Выходи
            <br />
            играть
          </p>
          <div className="mt-12 flex flex-wrap items-center justify-between gap-x-8 gap-y-6">
            <p className="max-w-[580px] text-[22px] leading-snug font-medium">
              Ничего подходящего рядом? Создайте свою встречу или соберите
              компанию из запросов.
            </p>
            <div className="flex flex-wrap gap-3">
              <Button
                variant="ink"
                size="xl"
                className="lift"
                render={<Link to={STATIC_ROUTES.EVENTS} />}
              >
                Найти встречу
              </Button>
              <Button
                variant="outline"
                size="xl"
                className="lift border-2 border-ink bg-transparent text-ink hover:bg-ink/10"
                render={<Link to={STATIC_ROUTES.EVENT_NEW} />}
              >
                Создать встречу
              </Button>
            </div>
          </div>
        </div>
      </section>
    </>
  );
};
