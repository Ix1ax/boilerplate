import type { Organization } from '@/entities/organization';
import { useSports } from '@/entities/sport';

import {
  Button,
  Checkbox,
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Notice,
  Textarea,
} from '@/shared/ui';

import { useClubForm } from '../model';

interface ClubFormProps {
  organization: Organization | null;
}

export const ClubForm = ({ organization }: ClubFormProps) => {
  const sports = useSports();
  const { form, save, submit } = useClubForm(organization);
  let submitLabel = organization ? 'Сохранить профиль' : 'Отправить заявку';
  if (organization?.status === 'rejected') submitLabel = 'Отправить повторно';
  if (save.isPending) submitLabel = 'Сохраняем…';
  return (
    <Form {...form}>
      <form
        onSubmit={submit}
        className="flex flex-col gap-4 rounded-3xl bg-card p-6"
      >
        <h2 className="title-display text-[26px]">
          {organization ? 'Профиль клуба' : 'Заявка клуба'}
        </h2>
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Название</FormLabel>
              <FormControl>
                <Input {...field} value={field.value ?? ''} />
              </FormControl>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="description"
          render={({ field }) => (
            <FormItem>
              <FormLabel>О клубе</FormLabel>
              <FormControl>
                <Textarea {...field} value={field.value ?? ''} />
              </FormControl>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="city"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Город</FormLabel>
              <FormControl>
                <Input {...field} value={field.value ?? ''} />
              </FormControl>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="contacts"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Контакт для администратора</FormLabel>
              <FormControl>
                <Textarea {...field} value={field.value ?? ''} />
              </FormControl>
              <FormDescription>Не публикуется в профиле клуба.</FormDescription>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="sport_ids"
          render={({ field }) => (
            <FormItem>
              <fieldset>
                <legend className="mb-2 font-medium">
                  Спортивные направления
                </legend>
                <div className="flex flex-wrap gap-x-5">
                  {sports.data?.map((sport) => (
                    <label
                      key={sport.id}
                      className="flex min-h-11 cursor-pointer items-center gap-2"
                    >
                      <Checkbox
                        checked={(field.value ?? []).includes(sport.id)}
                        onCheckedChange={(checked) =>
                          field.onChange(
                            checked
                              ? [...(field.value ?? []), sport.id]
                              : (field.value ?? []).filter(
                                  (id) => id !== sport.id,
                                ),
                          )
                        }
                      />
                      {sport.name}
                    </label>
                  ))}
                </div>
              </fieldset>
              <FormDescription>
                Выберите до 20 направлений, которые развивает клуб.
              </FormDescription>
              <FormMessage role="alert" />
              {sports.isPending && <p role="status">Загружаем направления…</p>}
              {sports.isError && (
                <Notice tone="warning">
                  Направления недоступны.
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => void sports.refetch()}
                  >
                    Повторить
                  </Button>
                </Notice>
              )}
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="site"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Сайт или страница — необязательно</FormLabel>
              <FormControl>
                <Input
                  {...field}
                  value={field.value ?? ''}
                  type="url"
                  placeholder="https://"
                />
              </FormControl>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="zones"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Районы, где проводите встречи</FormLabel>
              <FormControl>
                <Textarea
                  value={field.value?.join('\n') ?? ''}
                  onChange={(event) =>
                    field.onChange(
                      event.target.value ? event.target.value.split('\n') : [],
                    )
                  }
                  onBlur={field.onBlur}
                  ref={field.ref}
                  name={field.name}
                  placeholder="Каждый район с новой строки"
                />
              </FormControl>
              <FormDescription>
                До 20 районов, каждый с новой строки.
              </FormDescription>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="venues"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Где проводите встречи</FormLabel>
              <FormControl>
                <Textarea
                  value={field.value?.join('\n') ?? ''}
                  onChange={(event) =>
                    field.onChange(
                      event.target.value ? event.target.value.split('\n') : [],
                    )
                  }
                  onBlur={field.onBlur}
                  ref={field.ref}
                  name={field.name}
                  placeholder="Название площадки и адрес"
                />
              </FormControl>
              <FormDescription>
                Укажите названия и адреса, каждое место с новой строки.
              </FormDescription>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="representative"
          render={({ field }) => (
            <FormItem>
              <label className="flex min-h-11 items-start gap-3 rounded-2xl border p-4">
                <Checkbox
                  checked={field.value}
                  onCheckedChange={(checked) =>
                    field.onChange(checked === true)
                  }
                />
                <span>
                  Я представляю это сообщество и отвечаю за встречи от его имени
                </span>
              </label>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        {save.isError && (
          <Notice tone="danger" role="alert">
            {save.error.message}
          </Notice>
        )}
        {save.isSuccess && (
          <Notice tone="success" role="status">
            {organization?.status === 'pending'
              ? 'Заявка отправлена на проверку.'
              : 'Профиль сохранён.'}
          </Notice>
        )}
        <Button
          className="self-start"
          size="lg"
          type="submit"
          disabled={save.isPending}
        >
          {submitLabel}
        </Button>
      </form>
    </Form>
  );
};
