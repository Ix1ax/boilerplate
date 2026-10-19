import { Link } from 'react-router';

import { EventCardSkeleton } from '@/entities/event';

import { STATIC_ROUTES } from '@/shared/constants';
import { Button, Notice, Skeleton } from '@/shared/ui';

export const OrganizerProfileLoading = () => (
  <div
    role="status"
    aria-busy="true"
    className="mx-auto w-full max-w-[1280px] px-4 pt-7 pb-18 sm:px-8"
  >
    <span className="sr-only">Загружаем профиль…</span>
    <Skeleton className="mb-7 h-44 rounded-3xl" />
    <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
      {[1, 2, 3].map((id) => (
        <EventCardSkeleton key={id} />
      ))}
    </div>
  </div>
);

interface OrganizerProfileErrorProps {
  missing: boolean;
  onRetry: () => void;
}

export const OrganizerProfileError = ({
  missing,
  onRetry,
}: OrganizerProfileErrorProps) => (
  <div className="mx-auto w-full max-w-[1280px] px-4 pt-7 pb-18 sm:px-8">
    <Notice role="alert" tone="danger" className="gap-3 rounded-3xl p-6">
      <h1 className="title-display text-[26px]">
        {missing ? 'Профиль не найден' : 'Не удалось загрузить профиль'}
      </h1>
      <p>
        {missing
          ? 'Возможно, ссылка устарела или профиль недоступен.'
          : 'Проверьте соединение и попробуйте ещё раз.'}
      </p>
      <div className="flex flex-wrap gap-3">
        {!missing && (
          <Button size="lg" onClick={onRetry}>
            Повторить
          </Button>
        )}
        <Button
          variant="outline"
          size="lg"
          render={<Link to={STATIC_ROUTES.EVENTS} />}
        >
          Все встречи
        </Button>
      </div>
    </Notice>
  </div>
);
