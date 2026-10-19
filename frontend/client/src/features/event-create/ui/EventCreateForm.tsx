import { levelLabels } from '@starter/contracts';

import { Button, Form, Notice } from '@/shared/ui';

import { useEventCreate } from '../model';
import { DemandHint } from './DemandHint';
import {
  EventCheckboxField,
  EventChoiceField,
  EventInputField,
  EventTextField,
} from './EventFields';

export const EventCreateForm = () => {
  const {
    form,
    submit,
    errors,
    isSubmitting,
    sports,
    sportsQuery,
    organization,
  } = useEventCreate();
  const sportId = form.watch('sport_id');
  const sportName = form.watch('sport_name');
  return (
    <Form {...form}>
      <form
        onSubmit={(event) => void submit(event)}
        className="mt-5 grid max-w-[880px] gap-6 rounded-3xl bg-card p-6 sm:p-8"
        noValidate
      >
        <EventInputField
          form={form}
          name={'title'}
          label={'Название встречи'}
        />
        <EventChoiceField
          form={form}
          name={'sport_id'}
          label={'Направление'}
          items={[
            { value: 'custom', label: 'Впишу своё направление ниже' },
            ...sports.map((sport) => ({
              value: sport.id,
              label: sport.name,
            })),
          ]}
        />
        <EventInputField
          form={form}
          name={'sport_name'}
          label={'Своё направление'}
        />
        {sportsQuery.isError && (
          <p role="alert">
            Справочник недоступен. Можно вписать направление вручную или{' '}
            <button
              type="button"
              className="underline"
              onClick={() => void sportsQuery.refetch()}
            >
              повторить загрузку
            </button>
            .
          </p>
        )}
        {'sport' in errors && (
          <Notice role="alert" tone="danger">
            Выберите направление или впишите своё (от 2 до 60 символов).
          </Notice>
        )}
        <DemandHint sportId={sportId} sportName={sportName} />
        <div className="grid gap-6 sm:grid-cols-2">
          <EventInputField
            form={form}
            name={'starts_at'}
            label={'Начало (время Красноярска)'}
            type={'datetime-local'}
          />
          <EventInputField
            form={form}
            name={'ends_at'}
            label={'Окончание (время Красноярска, необязательно)'}
            type={'datetime-local'}
          />
        </div>
        <EventInputField
          form={form}
          name={'location'}
          label={'Место и адрес'}
        />
        <EventTextField form={form} name={'description'} label={'О встрече'} />
        <div className="grid gap-6 sm:grid-cols-2">
          <EventChoiceField
            form={form}
            name={'level'}
            label={'Уровень участников'}
            items={Object.entries(levelLabels).map(([value, label]) => ({
              value,
              label,
            }))}
          />
          <EventChoiceField
            form={form}
            name={'join_mode'}
            label={'Режим записи'}
            items={[
              { value: 'open', label: 'Свободная запись' },
              { value: 'request', label: 'По заявке' },
            ]}
          />
        </div>
        <EventTextField
          form={form}
          name={'requirements'}
          label={'Что взять и условия участия'}
        />
        <EventTextField
          form={form}
          name={'private_info'}
          label={'Точка сбора (только подтверждённым участникам)'}
        />
        <EventInputField
          form={form}
          name={'price_note'}
          label={'Стоимость / бесплатно'}
        />
        <div className="grid gap-6 sm:grid-cols-2">
          <EventInputField
            form={form}
            name={'capacity'}
            label={'Количество мест (пусто — без лимита)'}
            type={'number'}
          />
          <EventInputField
            form={form}
            name={'min_participants'}
            label={'Минимум для встречи (необязательно)'}
            type={'number'}
          />
        </div>
        <EventInputField
          form={form}
          name={'quorum_deadline'}
          label={'Срок сбора состава (время Красноярска, необязательно)'}
          type={'datetime-local'}
        />
        <EventCheckboxField form={form} name={'outdoor'} label={'На улице'} />
        {organization.data?.status === 'approved' && (
          <EventCheckboxField
            form={form}
            name={'as_organization'}
            label={`От имени клуба «${organization.data.name}»`}
          />
        )}
        <p className="text-sm text-muted-foreground">
          Для публикации от клуба его заявка должна быть одобрена в кабинете
          клуба.
        </p>
        {errors.root && (
          <Notice role="alert" tone="danger">
            {errors.root.message}
          </Notice>
        )}
        <Button type="submit" size="lg" disabled={isSubmitting}>
          {isSubmitting ? 'Публикуем…' : 'Опубликовать встречу'}
        </Button>
      </form>
    </Form>
  );
};
