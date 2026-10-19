import { levelLabels } from '@starter/contracts';

import type { CompanyRequest } from '@/entities/company';
import { useSports } from '@/entities/sport';

import { Button, Checkbox, Input, Notice, Textarea } from '@/shared/ui';

import { useCompanyRequestForm } from '../model';

interface CompanyRequestFormProps {
  onCreated: (request: CompanyRequest) => void;
  editing?: CompanyRequest;
  onCancel?: () => void;
}
const cityInput = (value: string) => {
  const date = new Date(value);
  return Number.isNaN(date.getTime())
    ? ''
    : new Date(date.getTime() + 7 * 3_600_000).toISOString().slice(0, 16);
};
const tomorrow = () => new Date(Date.now() + 86_400_000);
export const CompanyRequestForm = ({
  onCreated,
  editing,
  onCancel,
}: CompanyRequestFormProps) => {
  const sports = useSports();
  const { form, create, submit } = useCompanyRequestForm(onCreated, editing);
  const windows = form.watch('windows') ?? [];
  const zones = form.watch('zones') ?? [];
  const saveLabel = editing ? 'Сохранить изменения' : 'Искать компанию';
  const addWindow = () => {
    const start = tomorrow();
    start.setUTCMinutes(0, 0, 0);
    form.setValue('windows', [
      ...windows,
      {
        starts_at: start.toISOString(),
        ends_at: new Date(start.getTime() + 3_600_000).toISOString(),
      },
    ]);
  };
  return (
    <form
      onSubmit={submit}
      className="flex flex-col gap-5 self-start rounded-3xl bg-card p-6"
    >
      <h2 className="title-display text-2xl">
        {editing ? 'Изменить запрос' : 'Новый запрос'}
      </h2>
      <label className="grid gap-2 font-semibold">
        Вид спорта
        <Input
          {...form.register('sport_name')}
          list="company-sports"
          placeholder="Например, шахматы"
          onChange={(e) => {
            form.setValue('sport_name', e.target.value);
            form.setValue('sport_id', undefined);
          }}
        />
      </label>
      <datalist id="company-sports">
        {sports.data?.map((s) => (
          <option key={s.id} value={s.name} />
        ))}
      </datalist>
      <fieldset>
        <legend className="mb-2 font-semibold">Зоны, куда готовы прийти</legend>
        <p className="text-sm text-muted-foreground">
          Это не адрес — только район площадок
        </p>
        <div className="flex flex-wrap gap-3">
          {['Центр', 'Север', 'У парка'].map((z) => (
            <label key={z} className="flex min-h-11 items-center gap-2">
              <Checkbox
                checked={zones.includes(z)}
                onCheckedChange={(on) =>
                  form.setValue(
                    'zones',
                    on ? [...zones, z] : zones.filter((v) => v !== z),
                    { shouldValidate: true },
                  )
                }
              />
              {z}
            </label>
          ))}
        </div>
      </fieldset>
      <fieldset className="grid gap-3">
        <legend className="mb-2 font-semibold">
          Когда удобно — конкретные окна
        </legend>
        {windows.map((w, i) => (
          <div
            key={i}
            className="grid gap-2 rounded-2xl border border-border p-3"
          >
            <label className="grid gap-1">
              С
              <Input
                type="datetime-local"
                value={cityInput(w.starts_at)}
                onChange={(e) => {
                  const next = [...windows];
                  next[i] = {
                    ...w,
                    starts_at: e.target.value
                      ? `${e.target.value}:00+07:00`
                      : '',
                  };
                  form.setValue('windows', next);
                }}
              />
            </label>
            <label className="grid gap-1">
              До
              <Input
                type="datetime-local"
                value={cityInput(w.ends_at)}
                onChange={(e) => {
                  const next = [...windows];
                  next[i] = {
                    ...w,
                    ends_at: e.target.value ? `${e.target.value}:00+07:00` : '',
                  };
                  form.setValue('windows', next);
                }}
              />
            </label>
            <Button
              variant="outline"
              type="button"
              onClick={() =>
                form.setValue(
                  'windows',
                  windows.filter((_, n) => n !== i),
                )
              }
            >
              Убрать окно
            </Button>
          </div>
        ))}
        <Button
          type="button"
          variant="outline"
          onClick={addWindow}
          disabled={windows.length >= 20}
        >
          + Ещё окно
        </Button>
        <p className="text-sm text-muted-foreground">
          Время Красноярска. Для общего варианта нужен хотя бы час.
        </p>
      </fieldset>
      <label className="grid gap-2 font-semibold">
        Формат
        <select
          className="min-h-11 rounded-xl border border-border bg-background px-3"
          {...form.register('level')}
        >
          {Object.entries(levelLabels).map(([v, l]) => (
            <option key={v} value={v}>
              {l}
            </option>
          ))}
        </select>
      </label>
      <label className="grid gap-2 font-semibold">
        Запрос действует до
        <Input
          type="datetime-local"
          value={cityInput(form.watch('expires_at'))}
          onChange={(e) =>
            form.setValue(
              'expires_at',
              e.target.value ? `${e.target.value}:00+07:00` : '',
            )
          }
        />
      </label>
      <label className="grid gap-2 font-semibold">
        Пожелания
        <Textarea {...form.register('note')} />
      </label>
      {Object.values(form.formState.errors).map((error, i) => (
        <Notice key={i} tone="danger" role="alert">
          {error?.message ?? 'Проверьте даты и длительность окон'}
        </Notice>
      ))}
      {create.isError && (
        <Notice tone="danger" role="alert">
          {create.error.message}
        </Notice>
      )}
      <Button size="lg" type="submit" disabled={create.isPending}>
        {create.isPending ? 'Сохраняем…' : saveLabel}
      </Button>
      {onCancel && (
        <Button type="button" variant="outline" onClick={onCancel}>
          Отменить редактирование
        </Button>
      )}
    </form>
  );
};
