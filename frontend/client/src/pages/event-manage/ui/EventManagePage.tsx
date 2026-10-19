import { Link, useParams } from 'react-router';

import { useConfirm } from '@/features/confirm';
import { EventManagement } from '@/features/event-management';

import { DYNAMIC_ROUTES } from '@/shared/constants';
import { HeroBand } from '@/shared/ui';

export const EventManagePage = () => {
  const { eventId = '' } = useParams();
  const confirm = useConfirm();

  const onConfirmCancel = () => {
    return confirm({
      title: 'Отменить встречу?',
      description:
        'Участники увидят указанную причину и получат уведомление. Отмену нельзя вернуть.',
      confirmText: 'Отменить встречу',
      cancelText: 'Оставить встречу',
      destructive: true,
    });
  };
  return (
    <>
      <HeroBand
        tone="ink"
        eyebrow="Кабинет организатора"
        title="Управление встречей"
        description="Заявки, состав и напоминания участникам вашей встречи."
      />
      <div className="mx-auto w-full max-w-[1280px] px-4 pt-7 pb-18 sm:px-8">
        <Link
          to={DYNAMIC_ROUTES.EVENT(eventId)}
          className="mb-5 inline-flex min-h-11 items-center font-bold"
        >
          ← К встрече
        </Link>
        <EventManagement eventId={eventId} onConfirmCancel={onConfirmCancel} />
      </div>
    </>
  );
};
