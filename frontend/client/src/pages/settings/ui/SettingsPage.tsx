import { useForm, useWatch } from 'react-hook-form';
import { Link } from 'react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  profileSettingsSchema,
  type ProfileSettingsValues,
} from '@starter/contracts';
import { authProviderLabels } from '@starter/contracts';

import { ChangePasswordForm } from '@/features/auth';
import { MaxConnectCard } from '@/features/notifications';
import { ThemeToggle } from '@/features/theme';

import {
  type Profile,
  useMyProfile,
  useUpdateMyProfile,
} from '@/entities/profile';
import { useSports } from '@/entities/sport';
import { useCurrentUser } from '@/entities/user';

import { DYNAMIC_ROUTES } from '@/shared/constants';
import {
  Button,
  Checkbox,
  HeroBand,
  Input,
  Notice,
  Textarea,
} from '@/shared/ui';

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

        <ProfileSettings />

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

const ProfileSettings = () => {
  const profile = useMyProfile();
  if (profile.isPending) return <p role="status">Загружаем профиль…</p>;
  if (profile.isError)
    return (
      <Notice role="alert" tone="danger">
        Не удалось загрузить профиль.{' '}
        <Button onClick={() => void profile.refetch()}>Повторить</Button>
      </Notice>
    );
  return <ProfileEditor key={profile.data.id} profile={profile.data} />;
};
interface ProfileEditorProps {
  profile: Profile;
}
const ProfileEditor = ({ profile }: ProfileEditorProps) => {
  const form = useForm<ProfileSettingsValues>({
    resolver: zodResolver(profileSettingsSchema),
    defaultValues: {
      about: profile.about,
      city: profile.city,
      level:
        profile.level === 'beginner' ||
        profile.level === 'amateur' ||
        profile.level === 'pro'
          ? profile.level
          : '',
      interests: profile.interests ?? [],
      preferences: {
        notify_reminders: profile.preferences?.notifyReminders ?? true,
        notify_decisions: profile.preferences?.notifyDecisions ?? true,
        notify_replies: profile.preferences?.notifyReplies ?? true,
        large_text: profile.preferences?.largeText ?? false,
        reduced_motion: profile.preferences?.reducedMotion ?? false,
      },
    },
  });
  const values = useWatch({ control: form.control }) as ProfileSettingsValues;
  const setValues = (next: ProfileSettingsValues) => {
    form.setValue('interests', next.interests, { shouldDirty: true });
    form.setValue('preferences', next.preferences, { shouldDirty: true });
  };
  const sports = useSports();
  const validation = Object.values(form.formState.errors)
    .map((issue) => issue.message)
    .filter(Boolean)
    .join('. ');
  const save = useUpdateMyProfile();
  return (
    <section className="grid gap-5 rounded-3xl bg-card p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="title-display text-2xl">О себе</h2>
        <Button
          variant="outline"
          render={<Link to={DYNAMIC_ROUTES.PERSON(profile.id)} />}
        >
          Как меня видят другие
        </Button>
      </div>
      <p>
        Имя: <b>{profile.name}</b>. Оно задаётся учётной записью.
      </p>
      <form
        className="grid gap-5"
        noValidate
        onSubmit={form.handleSubmit((data) => save.mutate(data))}
      >
        <label className="grid gap-2">
          Город
          <Input
            autoComplete="address-level2"
            maxLength={100}
            {...form.register('city')}
          />
        </label>
        <label className="grid gap-2">
          Уровень подготовки
          <select
            className="min-h-11 rounded-xl border bg-background px-3"
            {...form.register('level')}
          >
            <option value="">Не указывать</option>
            <option value="beginner">Новичок</option>
            <option value="amateur">Любитель</option>
            <option value="pro">Профессионал</option>
          </select>
        </label>
        <label className="grid gap-2">
          Пару слов о себе — необязательно
          <Textarea maxLength={1000} {...form.register('about')} />
          <span className="text-sm text-muted-foreground">
            Видно в вашем публичном профиле.
          </span>
        </label>
        <fieldset className="grid gap-3">
          <legend className="mb-2 font-medium">Что интересно</legend>
          <div className="flex flex-wrap gap-4">
            {sports.data?.map((sport) => (
              <label
                key={sport.id}
                className="flex min-h-11 items-center gap-2"
              >
                <Checkbox
                  checked={values.interests.includes(sport.name)}
                  onCheckedChange={(checked) =>
                    setValues({
                      ...values,
                      interests: checked
                        ? [...values.interests, sport.name]
                        : values.interests.filter(
                            (value) => value !== sport.name,
                          ),
                    })
                  }
                />
                {sport.name}
              </label>
            ))}
          </div>
          {sports.isError && (
            <p>
              Не удалось загрузить направления.{' '}
              <Button type="button" onClick={() => void sports.refetch()}>
                Повторить
              </Button>
            </p>
          )}
        </fieldset>
        <fieldset className="grid gap-3">
          <legend className="mb-2 font-medium">
            Какие уведомления получать
          </legend>
          {(
            [
              { key: 'notify_reminders', label: 'Напоминания о встречах' },
              {
                key: 'notify_decisions',
                label: 'Решения и изменения моих встреч и заявок',
              },
              { key: 'notify_replies', label: 'Вопросы и ответы в обсуждении' },
            ] as const
          ).map(({ key, label }) => (
            <label key={key} className="flex min-h-11 items-center gap-3">
              <Checkbox
                checked={values.preferences[key]}
                onCheckedChange={(checked) =>
                  setValues({
                    ...values,
                    preferences: {
                      ...values.preferences,
                      [key]: checked === true,
                    },
                  })
                }
              />
              {label}
            </label>
          ))}
          <p className="text-sm text-muted-foreground">
            Для новых уведомлений во всех каналах: колокольчик и MAX.
            Существующие уведомления сохраняются.
          </p>
        </fieldset>
        <fieldset className="grid gap-3">
          <legend className="mb-2 font-medium">Удобство</legend>
          <label className="flex min-h-11 items-center gap-3">
            <Checkbox
              checked={values.preferences.large_text}
              onCheckedChange={(checked) =>
                setValues({
                  ...values,
                  preferences: {
                    ...values.preferences,
                    large_text: checked === true,
                  },
                })
              }
            />
            Крупный текст
          </label>
          <label className="flex min-h-11 items-center gap-3">
            <Checkbox
              checked={values.preferences.reduced_motion}
              onCheckedChange={(checked) =>
                setValues({
                  ...values,
                  preferences: {
                    ...values.preferences,
                    reduced_motion: checked === true,
                  },
                })
              }
            />
            Меньше движения
          </label>
          <p className="text-sm text-muted-foreground">
            Настройки применяются ко всему сервису после сохранения.
          </p>
        </fieldset>
        {(validation || save.isError) && (
          <Notice role="alert" tone="danger">
            {validation || save.error?.message} Ваши изменения сохранены в
            форме.
          </Notice>
        )}
        {save.isSuccess && (
          <Notice role="status" tone="success">
            Профиль сохранён.
          </Notice>
        )}
        <Button
          type="submit"
          className="justify-self-start"
          disabled={save.isPending}
        >
          {save.isPending ? 'Сохраняем…' : 'Сохранить профиль'}
        </Button>
      </form>
    </section>
  );
};
