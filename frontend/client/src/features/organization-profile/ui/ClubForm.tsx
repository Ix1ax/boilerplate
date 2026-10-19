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
              <FormLabel>Контакты</FormLabel>
              <FormControl>
                <Textarea {...field} value={field.value ?? ''} />
              </FormControl>
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
