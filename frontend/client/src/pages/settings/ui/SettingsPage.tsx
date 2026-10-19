import { authProviderLabels } from '@starter/contracts';

import { ChangePasswordForm } from '@/features/auth';
import { MaxConnectCard } from '@/features/notifications';
import { ThemeToggle } from '@/features/theme';

import { useCurrentUser } from '@/entities/user';

import { HeroBand } from '@/shared/ui';

export const SettingsPage = () => {
  const { data: user, isPending } = useCurrentUser();

  return (
    <>
      <HeroBand
        eyebrow="ваш аккаунт"
        title="Профиль и настройки"
        description="Способ входа, пароль, уведомления и внешний вид интерфейса."
      />
      <div className="mx-auto flex w-full max-w-[880px] flex-col gap-10 px-4 pt-7 pb-18 sm:px-8">
        {!isPending && user && (
          <section className="flex flex-col gap-2">
            <h2 className="title-display text-2xl">Аккаунт</h2>
            <p>
              <b>{user.name || user.email}</b>{' '}
              <span className="text-muted-foreground">· {user.email}</span>
            </p>
            <p className="text-sm text-muted-foreground">
              Способ входа: {authProviderLabels[user.provider]}
            </p>
          </section>
        )}

        {user?.provider === 'local' && (
          <section className="flex flex-col gap-4">
            <h2 className="title-display text-2xl">Пароль</h2>
            <ChangePasswordForm />
          </section>
        )}

        <section className="flex flex-col gap-4">
          <h2 className="title-display text-2xl">Уведомления</h2>
          <MaxConnectCard />
        </section>

        <section className="flex flex-col gap-4">
          <h2 className="title-display text-2xl">Внешний вид</h2>
          <div className="flex items-center justify-between gap-4 rounded-2xl bg-card p-5">
            <div>
              <b>Тема оформления</b>
              <p className="mt-1 text-sm text-muted-foreground">
                Светлая, тёмная или по настройкам системы.
              </p>
            </div>
            <ThemeToggle />
          </div>
        </section>
      </div>
    </>
  );
};
