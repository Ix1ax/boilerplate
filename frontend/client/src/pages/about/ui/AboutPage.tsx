import { Link } from 'react-router';

import { STATIC_ROUTES } from '@/shared/constants';
import { Button, Eyebrow, RadarRings } from '@/shared/ui';

const STEPS = [
  ['01', 'Найдите встречу', 'Списком: спорт, дата, режим записи.'],
  [
    '02',
    'Спросите до записи',
    'Опыт, инвентарь, кто встретит. Ответ организатора выделен.',
  ],
  [
    '03',
    'Запишитесь',
    'Свободно — сразу в составе. По заявке — решает организатор.',
  ],
  [
    '04',
    'Нет встречи — соберите',
    'Запрос → подходящая встреча → вы записываетесь сами.',
  ],
];

const STATUSES = [
  [
    'Запрос',
    'Вы сохранили спорт, район и время',
    '«Запрос сохранён»',
    'border-[1.5px] border-dashed border-muted-foreground bg-card',
  ],
  [
    'Интерес',
    'Вам подходит найденная встреча',
    '«Это ещё не запись»',
    'bg-interest-soft text-interest',
  ],
  [
    'Заявка',
    'Вы попросили принять вас',
    '«Ожидает решения организатора»',
    'bg-warning-soft text-warning',
  ],
  [
    'Участие',
    'Свободная запись или одобрение',
    '«Вы записаны»',
    'bg-success-soft text-success',
  ],
];

export const AboutPage = () => (
  <>
    <section className="relative overflow-hidden bg-ink text-paper">
      <RadarRings
        tone="gold"
        sizes={[240, 460, 700]}
        strength={0.55}
        className="top-1/2 right-[230px] max-md:right-0"
      />
      <div className="relative mx-auto max-w-[1280px] px-4 pt-14 pb-20 sm:px-8">
        <Eyebrow className="text-gold">о сервисе</Eyebrow>
        <h1 className="display-caps mt-3.5 max-w-[1000px] text-[clamp(40px,6vw,88px)] leading-[.9]">
          От «не&nbsp;с&nbsp;кем» — к&nbsp;встрече, на которую понятно как
          прийти
        </h1>
        <p className="mt-6 max-w-[680px] text-xl text-paper/90">
          «Вместе» помогает найти людей для любительского спорта рядом с домом —
          от зарядки во дворе до шахмат. Встречи создают сами люди и допущенные
          клубы.
        </p>
      </div>
    </section>

    <div className="mx-auto flex w-full max-w-[1280px] flex-col gap-16 px-4 pt-16 pb-18 sm:px-8">
      <section className="flex flex-col gap-5">
        <h2 className="display-caps text-4xl">Как это работает</h2>
        <ol className="grid grid-cols-[repeat(auto-fit,minmax(min(250px,100%),1fr))] gap-3.5">
          {STEPS.map(([n, t, d], i) => (
            <li
              key={n}
              className={`flex flex-col gap-2.5 rounded-3xl p-6 ${i === 3 ? 'bg-blue text-white' : 'bg-card'}`}
            >
              <span
                className={`font-display text-[44px] leading-none font-black ${i === 3 ? 'text-gold' : 'text-blue dark:text-interest'}`}
              >
                {n}
              </span>
              <b className="text-xl">{t}</b>
              <span>{d}</span>
            </li>
          ))}
        </ol>
      </section>

      <section className="flex flex-col gap-5">
        <h2 className="display-caps text-4xl">Четыре разных статуса</h2>
        <p className="max-w-[760px]">
          Мы не смешиваем их и не складываем в одно число. Встреча, созданная по
          вашему запросу, никого не записывает автоматически.
        </p>
        <div className="grid grid-cols-[repeat(auto-fit,minmax(min(250px,100%),1fr))] gap-3">
          {STATUSES.map(([t, d, s, cls]) => (
            <div
              key={t}
              className={`flex flex-col gap-1.5 rounded-[22px] p-5 ${cls}`}
            >
              <b className="text-[19px]">{t}</b>
              <span>{d}</span>
              <span className="font-bold">{s}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="grid grid-cols-[repeat(auto-fit,minmax(min(380px,100%),1fr))] gap-4">
        <div className="flex flex-col gap-2.5 rounded-[28px] bg-card p-7">
          <h2 className="title-display text-2xl">Для клубов</h2>
          <span>
            Заявка → проверка администратором → публикация встреч без проверки
            каждой. При отказе причина видна, заявку можно исправить.
          </span>
        </div>
        <div className="flex flex-col gap-2.5 rounded-[28px] bg-card p-7">
          <h2 className="title-display text-2xl">Чего в сервисе нет</h2>
          <ul className="list-disc pl-5">
            <li>бронирования площадок и оплаты;</li>
            <li>рейтингов людей;</li>
            <li>личной переписки — только обсуждение встречи;</li>
            <li>гарантий безопасности — встречи проводят люди и клубы.</li>
          </ul>
        </div>
      </section>

      <section className="relative flex flex-wrap items-center justify-between gap-6 overflow-hidden rounded-[32px] bg-gold p-9 text-ink">
        <div className="flex max-w-[760px] flex-col gap-2.5">
          <h2 className="display-caps text-[26px]">Учебный статус</h2>
          <p className="text-xl font-bold">
            Учебный прототип по кейсу Столото. Не является действующим сервисом
            компании.
          </p>
          <p>Имена, клубы и площадки вымышлены, числа — демонстрационные.</p>
        </div>
        <Button
          variant="ink"
          size="lg"
          render={<Link to={STATIC_ROUTES.EVENTS} />}
        >
          Перейти к встречам
        </Button>
      </section>
    </div>
  </>
);
