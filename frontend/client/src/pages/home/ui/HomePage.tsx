import { Link } from 'react-router';
import { ArrowRightIcon } from 'lucide-react';
import { motion } from 'motion/react';

import { EventCard, useEvents } from '@/entities/event';
import { useSports } from '@/entities/sport';
import { useCurrentUser } from '@/entities/user';

import { STATIC_ROUTES } from '@/shared/constants';
import { useSmoothScroll } from '@/shared/hooks';
import {
  DURATION,
  EASING,
  Lift,
  Magnetic,
  MaskRise,
  Parallax,
  Reveal,
  staggerDelay,
} from '@/shared/motion';
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

const PINGS = [
  { className: 'top-[24%] left-[83%] size-3 bg-gold' },
  { className: 'top-[61%] left-[13%] size-2.5 bg-paper' },
  { className: 'top-[89%] left-[66%] size-3 bg-gold' },
  { className: 'top-[8%] left-[30%] size-[9px] bg-paper' },
];

// Декоративные карточки радара: демонстрационные примеры, не данные из API.
const FLOATERS = [
  {
    title: 'Шахматы без опыта',
    meta: 'вт 20.10 · 12:00 · Центр',
    className: 'top-[6%] left-[56%] bg-white text-ink',
    metaClassName: 'text-[#4A505B]',
  },
  {
    title: 'Дворовая зарядка',
    meta: 'вт 20.10 · 09:00 · У парка',
    className: 'top-[30%] -left-[4%] bg-gold text-ink',
    metaClassName: 'text-[#3D3200]',
  },
  {
    title: 'Игровая тренировка',
    meta: 'ср 21.10 · по заявке',
    className: 'top-[68%] left-[54%] bg-blue text-white',
    metaClassName: 'text-white/85',
  },
  {
    title: '4 запроса · соберём?',
    meta: 'шахматы · вечер · будни',
    className:
      'top-[80%] left-[2%] border-[1.5px] border-dashed border-gold bg-night text-gold',
    metaClassName: 'text-paper/80',
  },
];

export const HomePage = () => {
  useSmoothScroll();
  // Лендинг открыт гостю: каталог и направления доступны только после входа, а любой 401
  // уводит на страницу входа. Поэтому гостю показываем лендинг без живых данных.
  const { data: user, isPending: userPending } = useCurrentUser();
  const signedIn = !userPending && Boolean(user);
  const { data } = useEvents({ limit: 6 }, { enabled: signedIn });
  const { data: allSports } = useSports({ enabled: signedIn });
  const sports = (allSports ?? []).filter((sport) => !sport.is_custom);
  const events = data?.items ?? [];

  return (
    <>
      <section className="relative overflow-hidden bg-ink bg-[linear-gradient(rgb(238_240_234/5%)_1px,transparent_1px),linear-gradient(90deg,rgb(238_240_234/5%)_1px,transparent_1px)] bg-[size:80px_80px] text-paper">
        <div className="relative mx-auto flex max-w-[1360px] flex-wrap items-center gap-10 px-4 pt-16 pb-24 sm:px-8 sm:pt-20 sm:pb-32">
          <div className="relative z-[2] min-w-0 flex-[1_1_620px]">
            <Reveal immediate delay={0.06} y={0} duration={0.48}>
              <Eyebrow className="flex flex-wrap gap-x-5 gap-y-2 text-paper/80">
                <span className="inline-flex items-center gap-2">
                  <span className="size-2 rounded-full bg-gold" />
                  Красноярск · демо-город
                </span>
                <span>любительский спорт рядом с домом</span>
              </Eyebrow>
            </Reveal>
            <h1 className="display-caps mt-5 text-[clamp(64px,11vw,172px)] leading-[.86] tracking-[-0.045em]">
              <MaskRise delay={0.12}>Найди</MaskRise>
              <MaskRise delay={0.2} className="text-gold">
                своих
              </MaskRise>
            </h1>
            <Reveal
              as="p"
              immediate
              delay={0.4}
              y={16}
              className="mt-6 max-w-[640px] text-[clamp(20px,1.9vw,25px)] leading-snug"
            >
              Компания для шахмат, бега, зарядки и вашего собственного вида
              спорта. А если подходящей встречи ещё нет — её можно собрать.
            </Reveal>
            <Reveal
              immediate
              delay={0.5}
              y={16}
              className="mt-9 flex flex-wrap gap-3"
            >
              <Magnetic>
                <Lift>
                  <Button
                    size="xl"

                    render={<Link to={STATIC_ROUTES.EVENTS} />}
                  >
                    Найти встречу
                    <ArrowRightIcon aria-hidden />
                  </Button>
                </Lift>
              </Magnetic>
              <Magnetic>
                <Lift>
                  <Button
                    size="xl"
                    variant="gold"

                    render={<Link to={STATIC_ROUTES.COMPANY} />}
                  >
                    Собрать компанию
                  </Button>
                </Lift>
              </Magnetic>
            </Reveal>
          </div>

          <Parallax speed={-0.15} className="min-w-[300px] flex-[0_1_540px]">
            <Reveal
              immediate
              delay={0.31}
              y={0}
              scale={0.94}
              duration={0.9}
              className="relative aspect-square"
            >
              <div aria-hidden="true" className="absolute inset-0">
                <RadarRings
                  tone="gold"
                  sizes={[280, 360, 420, 480, 540]}
                  strength={0.5}
                  className="top-1/2 left-1/2"
                />
                <div className="absolute inset-[40%] grid place-items-center rounded-full bg-blue">
                  <span className="relative block aspect-[1.5] w-[52%]">
                    <span className="absolute top-0 left-0 aspect-square h-full rounded-full bg-white" />
                    <span className="absolute top-0 right-0 aspect-square h-full rounded-full bg-gold shadow-[0_0_0_3px_var(--color-blue)]" />
                  </span>
                </div>
                {PINGS.map((ping, index) => (
                  <motion.span
                    key={ping.className}
                    className={`absolute rounded-full ${ping.className}`}
                    animate={{ scale: [1, 1.7, 1], opacity: [1, 0.55, 1] }}
                    transition={{
                      duration: 3.2,
                      delay: index * 0.7,
                      ease: EASING.inOut,
                      repeat: Infinity,
                    }}
                  />
                ))}
                {FLOATERS.map((card, index) => (
                  <motion.div
                    key={card.title}
                    className={`absolute rounded-[18px] px-4 py-3 whitespace-nowrap shadow-[0_20px_40px_-14px_rgb(0_0_0/60%)] ${card.className}`}
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: [0, -8, 0] }}
                    transition={{
                      opacity: {
                        delay: 0.6 + index * 0.12,
                        duration: DURATION.slow,
                      },
                      y: {
                        delay: 0.6 + index * 0.12,
                        duration: 5 + index,
                        ease: EASING.inOut,
                        repeat: Infinity,
                      },
                    }}
                  >
                    <div className="text-base font-bold">{card.title}</div>
                    <div
                      className={`font-mono text-[13px] ${card.metaClassName}`}
                    >
                      {card.meta}
                    </div>
                  </motion.div>
                ))}
              </div>
            </Reveal>
          </Parallax>
        </div>

        {sports.length > 0 && (
          <div className="mt-2 border-t border-paper/15">
            <ul
              aria-label="Виды спорта на платформе"
              className="mx-auto flex max-w-[1360px] flex-wrap items-center gap-x-7 gap-y-2.5 px-4 py-5 font-display text-[clamp(17px,1.8vw,24px)] font-extrabold tracking-[-0.01em] uppercase sm:px-8"
            >
              {sports.map((sport, index) => (
                <Reveal
                  as="li"
                  key={sport.id}
                  immediate
                  delay={0.7 + staggerDelay(index) * 0.5}
                  y={12}
                  className="flex items-center gap-7"
                >
                  {sport.name}
                  <span
                    aria-hidden="true"
                    className="size-2 rounded-full bg-gold"
                  />
                </Reveal>
              ))}
              <li>
                <Link
                  to={STATIC_ROUTES.EVENT_NEW}
                  className="inline-flex min-h-11 items-center text-gold"
                >
                  + свой вид
                </Link>
              </li>
            </ul>
          </div>
        )}
      </section>

      <section
        id="path"
        className="bg-paper px-4 py-24 text-ink sm:px-8 sm:py-28"
      >
        <div className="mx-auto max-w-[1360px]">
          <div className="flex flex-wrap items-end justify-between gap-x-12 gap-y-6">
            <h2 className="display-caps max-w-[920px] text-[clamp(32px,5.6vw,84px)] leading-[.92] tracking-[-0.035em]">
              <MaskRise inView>
                От «не&nbsp;с&nbsp;кем» до&nbsp;«я&nbsp;в&nbsp;деле»
              </MaskRise>
            </h2>
            <p className="max-w-[400px] text-[19px] text-ink-soft">
              Одна логика для любого спорта: найти, понять условия, спросить,
              записаться и прийти.
            </p>
          </div>
          <ol className="mt-16 grid grid-cols-1 gap-x-10 gap-y-10 border-t-2 border-ink sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
            {STEPS.map(([n, title, text], index) => (
              <Reveal
                as="li"
                key={n}
                delay={staggerDelay(index)}
                className="flex flex-col gap-4 border-b border-line-strong pt-8 pb-10"
              >
                <span className="font-display text-[56px] leading-none font-black text-blue">
                  {n}
                </span>
                <b className="text-[21px] leading-snug">{title}</b>
                <span className="text-base leading-relaxed text-ink-soft">
                  {text}
                </span>
              </Reveal>
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
              <MaskRise inView>
                Рядом
                <br />и скоро
              </MaskRise>
            </h2>
          </div>
          <Lift>
            <Button
              variant="gold"
              size="xl"

              render={<Link to={STATIC_ROUTES.EVENTS} />}
            >
              Все встречи
              <ArrowRightIcon aria-hidden />
            </Button>
          </Lift>
        </div>
        {events.length > 0 && (
          <ul className="relative mt-14 flex snap-x snap-mandatory gap-5 overflow-x-auto px-4 pt-3 pb-7 sm:px-8">
            {events.map((event, index) => (
              <Reveal
                as="li"
                key={event.id}
                delay={staggerDelay(index)}
                className="flex w-80 shrink-0 snap-start"
              >
                <EventCard event={event} className="w-full" />
              </Reveal>
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
            <MaskRise inView>Нет встречи? Соберём её из&nbsp;спроса</MaskRise>
          </h2>
          <ol className="mt-16 grid grid-cols-[repeat(auto-fit,minmax(min(260px,100%),1fr))] gap-5">
            {TOGETHER.map(([title, text, tag], i) => (
              <Reveal
                as="li"
                key={title}
                delay={staggerDelay(i)}
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
              </Reveal>
            ))}
          </ol>
          <div className="mt-7 flex flex-wrap items-center justify-between gap-5 rounded-3xl bg-gold px-7 py-6 text-ink">
            <p className="max-w-[760px] text-[19px] leading-snug font-semibold">
              Интерес — ещё не запись. Публикация встречи никого не добавляет в
              состав автоматически, а организатор не видит ваших контактов.
            </p>
            <Lift>
              <Button
                variant="ink"
                size="lg"

                render={<Link to={STATIC_ROUTES.COMPANY} />}
              >
                Собрать компанию
              </Button>
            </Lift>
          </div>
        </div>
      </section>

      <section id="start" className="bg-paper px-4 py-28 text-ink sm:px-8">
        <div className="mx-auto max-w-[1360px]">
          <Eyebrow className="text-blue">новичкам и старшему поколению</Eyebrow>
          <h2 className="display-caps mt-3.5 max-w-[900px] text-[clamp(40px,5.2vw,80px)] leading-[.92] tracking-[-0.035em]">
            <MaskRise inView>Не страшно прийти впервые</MaskRise>
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
            <MaskRise inView>Собирайте своё сообщество</MaskRise>
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
              <Reveal
                key={k}
                delay={staggerDelay(i)}
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
              </Reveal>
            ))}
          </div>
          <Lift className="mt-10 inline-flex">
            <Button
              variant="ink"
              size="xl"
              render={<Link to={STATIC_ROUTES.CLUB} />}
            >
              Подать заявку клуба
            </Button>
          </Lift>
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
            <MaskRise inView>
              Выходи
              <br />
              играть
            </MaskRise>
          </p>
          <div className="mt-12 flex flex-wrap items-center justify-between gap-x-8 gap-y-6">
            <p className="max-w-[580px] text-[22px] leading-snug font-medium">
              Ничего подходящего рядом? Создайте свою встречу или соберите
              компанию из запросов.
            </p>
            <div className="flex flex-wrap gap-3">
              <Lift>
                <Button
                  variant="ink"
                  size="xl"

                  render={<Link to={STATIC_ROUTES.EVENTS} />}
                >
                  Найти встречу
                </Button>
              </Lift>
              <Lift>
                <Button
                  variant="outline"
                  size="xl"
                  className="border-2 border-ink bg-transparent text-ink hover:bg-ink/10"
                  render={<Link to={STATIC_ROUTES.EVENT_NEW} />}
                >
                  Создать встречу
                </Button>
              </Lift>
            </div>
          </div>
        </div>
      </section>
    </>
  );
};
