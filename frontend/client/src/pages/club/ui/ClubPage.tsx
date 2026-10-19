import { ClubForm } from '@/features/organization-profile';

import { useMyOrganization } from '@/entities/organization';

import { Button, HeroBand, Notice } from '@/shared/ui';

import { ClubStatus } from './ClubStatus';

export const ClubPage = () => {
  const organization = useMyOrganization();

  return (
    <>
      <HeroBand
        title="Кабинет клуба"
        eyebrow="организаторам"
        description="Представьте свой клуб. После проверки можно создавать встречи от имени организации."
      />
      <div className="mx-auto flex w-full max-w-[1280px] flex-col gap-5 px-4 pt-7 pb-18 sm:px-8">
        {organization.isPending && (
          <p role="status">Загружаем профиль клуба…</p>
        )}
        {organization.isError && (
          <Notice tone="danger" role="alert">
            Не удалось загрузить профиль.
            <Button onClick={() => void organization.refetch()}>
              Повторить
            </Button>
          </Notice>
        )}
        {!organization.isPending && !organization.isError && (
          <>
            {organization.data && (
              <ClubStatus organization={organization.data} />
            )}
            <ClubForm organization={organization.data ?? null} />
          </>
        )}
      </div>
    </>
  );
};
