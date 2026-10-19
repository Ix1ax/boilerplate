import { useState } from 'react';
import { Link } from 'react-router';
import { levelLabels } from '@starter/contracts';

import { LocationField } from '@/features/location-picker';

import { formatWhen } from '@/entities/event';

import { DYNAMIC_ROUTES } from '@/shared/constants';
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
    created,
    submit,
    errors,
    isSubmitting,
    sports,
    sportsQuery,
    organization,
  } = useEventCreate();
  const [step, setStep] = useState(0);
  const values = form.watch();
  const sportId = form.watch('sport_id');
  const sportName = form.watch('sport_name');
  if (created)
    return (
      <section
        className="mt-6 grid max-w-3xl gap-5 rounded-3xl bg-card p-8"
        role="status"
      >
        <h2 className="title-display text-3xl">Встреча опубликована</h2>
        <p>Участники уже могут найти «{created.title}» в каталоге.</p>
        <div className="flex flex-wrap gap-3">
          <Button render={<Link to={DYNAMIC_ROUTES.EVENT(created.id)} />}>
            Открыть встречу
          </Button>
          <Button
            variant="outline"
            render={<Link to={DYNAMIC_ROUTES.EVENT_MANAGE(created.id)} />}
          >
            Управлять встречей
          </Button>
        </div>
      </section>
    );
  const next = async () => {
    const valid = await form.trigger(
      step === 0
        ? [
            'title',
            'sport_id',
            'sport_name',
            'starts_at',
            'ends_at',
            'location',
            'lat',
            'lng',
            'zone',
          ]
        : [
            'description',
            'price_note',
            'meeting_host',
            'outdoor',
            'step_free',
            'companion_allowed',
            'as_organization',
            'level',
            'join_mode',
            'requirements',
            'private_info',
            'capacity',
            'min_participants',
            'quorum_deadline',
          ],
      { shouldFocus: true },
    );
    if (step === 0 && !sportId && !sportName?.trim()) {
      form.setError('sport_name', {
        message: 'Выберите направление или впишите своё',
      });
      return;
    }
    if (valid) setStep((value) => Math.min(2, value + 1));
  };
  return (
    <Form {...form}>
      <form
        onSubmit={(event) => {
          if (step < 2) {
            event.preventDefault();
            void next();
          } else void submit(event);
        }}
        className="mt-5 grid max-w-[880px] gap-6 rounded-3xl bg-card p-6 sm:p-8"
        noValidate
      >
        <nav aria-label="Этапы создания встречи">
          <ol className="flex flex-wrap gap-3">
            {['Где и когда', 'Участие и условия', 'Проверка'].map(
              (label, index) => (
                <li
                  key={label}
                  aria-current={index === step ? 'step' : undefined}
                  className={
                    index === step
                      ? 'font-bold text-blue'
                      : 'text-muted-foreground'
                  }
                >
                  {index + 1}. {label}
                </li>
              ),
            )}
          </ol>
        </nav>
        <fieldset hidden={step !== 0} className="grid gap-6">
          <legend className="sr-only">Где и когда</legend>
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
          <LocationField
            lat={form.watch('lat')}
            lng={form.watch('lng')}
            onChange={({ lat, lng, address }) => {
              form.setValue('lat', lat);
              form.setValue('lng', lng);
              if (address) form.setValue('location', address);
            }}
          />
          <EventInputField form={form} name="zone" label="Зона / район" />
        </fieldset>
        <fieldset hidden={step !== 1} className="grid gap-6">
          <legend className="sr-only">Участие и условия</legend>
          <EventTextField
            form={form}
            name={'description'}
            label={'О встрече'}
          />
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
          <EventInputField
            form={form}
            name="meeting_host"
            label="Кто встретит участников"
          />
          <EventCheckboxField
            form={form}
            name="step_free"
            label="Без ступеней / есть пандус"
          />
          <EventCheckboxField
            form={form}
            name="companion_allowed"
            label="Можно с сопровождающим"
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
        </fieldset>
        {step === 2 && (
          <section
            aria-label="Предпросмотр встречи"
            className="grid gap-4 rounded-3xl border border-border p-6"
          >
            <h2 className="title-display text-2xl">Проверьте встречу</h2>
            <h3 className="text-xl font-bold">{values.title}</h3>
            <p>
              {sports.find((sport) => sport.id === sportId)?.name || sportName}
            </p>
            <p>
              {values.starts_at &&
                Number.isFinite(new Date(values.starts_at).getTime()) &&
                formatWhen(values.starts_at)}
            </p>
            <p>
              {values.location} · {values.zone}
            </p>
            <p className="whitespace-pre-line">{values.description}</p>
            <p>
              {values.join_mode === 'request'
                ? 'По заявке с подтверждением'
                : 'Свободная запись'}{' '}
              ·{' '}
              {values.capacity ? `${values.capacity} мест` : 'Без лимита мест'}
            </p>
            <p>{values.requirements || 'Особые условия не указаны'}</p>
            <p>{values.price_note || 'Стоимость не указана'}</p>
            <p>
              {values.as_organization
                ? `От клуба «${organization.data?.name}»`
                : 'От моего имени'}
            </p>
            <p className="text-sm text-muted-foreground">
              Точка сбора доступна только подтверждённым участникам. После
              публикации встречу можно изменить в управлении.
            </p>
          </section>
        )}
        {Object.keys(errors).some((key) => key !== 'root') && (
          <Notice tone="danger" role="alert">
            Проверьте поля формы.{' '}
            <Button type="button" variant="link" onClick={() => setStep(0)}>
              Вернуться к началу
            </Button>
          </Notice>
        )}
        {errors.root && (
          <Notice role="alert" tone="danger">
            {errors.root.message}
          </Notice>
        )}
        <div className="flex flex-wrap gap-3">
          {step > 0 && (
            <Button
              type="button"
              variant="outline"
              onClick={() => setStep((value) => value - 1)}
            >
              Назад
            </Button>
          )}
          {step < 2 ? (
            <Button type="button" onClick={() => void next()}>
              Далее
            </Button>
          ) : (
            <Button type="submit" size="lg" disabled={isSubmitting}>
              {isSubmitting ? 'Публикуем…' : 'Опубликовать встречу'}
            </Button>
          )}
        </div>
      </form>
    </Form>
  );
};
