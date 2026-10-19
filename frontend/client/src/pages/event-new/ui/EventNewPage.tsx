import { Link } from 'react-router';

import { EventCreateForm } from '@/features/event-create';

import { STATIC_ROUTES } from '@/shared/constants';
import { HeroBand } from '@/shared/ui';

export const EventNewPage = () => {
  return (
    <>
      <HeroBand
        tone="blue"
        eyebrow="Ваша спортивная встреча"
        title="Создать встречу"
        description="Укажите место, время и условия — участники смогут найти встречу сразу после публикации."
      />
      <div className="mx-auto w-full max-w-[1280px] px-4 pt-7 pb-18 sm:px-8">
        <Link
          to={STATIC_ROUTES.EVENTS}
          className="inline-flex min-h-11 items-center font-bold"
        >
          ← Все встречи
        </Link>
        <EventCreateForm />
      </div>
    </>
  );
};
