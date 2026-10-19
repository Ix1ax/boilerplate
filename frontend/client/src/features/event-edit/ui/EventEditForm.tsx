import { type Event,levelLabels } from '@starter/contracts';

import {
  Button,
  Checkbox,
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Notice,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Textarea,
} from '@/shared/ui';

import { useEventEdit } from '../model';

interface EventEditFormProps {
  event: Event;
}

const LEVEL_ITEMS = Object.entries(levelLabels).map(([value, label]) => ({
  value,
  label,
}));

/** Правка опубликованной, ещё не начавшейся встречи: направление и режим записи не меняются. */
export const EventEditForm = ({ event }: EventEditFormProps) => {
  const { form, submit, errors, isSubmitting, isSubmitSuccessful } =
    useEventEdit(event);

  return (
    <Form {...form}>
      <form
        onSubmit={(domEvent) => void submit(domEvent)}
        className="grid gap-6 rounded-3xl bg-card p-6 sm:p-8"
        noValidate
      >
        <FormField
          control={form.control}
          name="title"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Название встречи</FormLabel>
              <FormControl>
                <Input {...field} value={field.value ?? ''} />
              </FormControl>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <div className="grid gap-6 sm:grid-cols-2">
          <FormField
            control={form.control}
            name="starts_at"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Начало (время Красноярска)</FormLabel>
                <FormControl>
                  <Input
                    type="datetime-local"
                    value={String(field.value ?? '').slice(0, 16)}
                    onChange={(e) =>
                      field.onChange(
                        e.target.value ? `${e.target.value}:00+07:00` : undefined,
                      )
                    }
                  />
                </FormControl>
                <FormMessage role="alert" />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="ends_at"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Окончание (необязательно)</FormLabel>
                <FormControl>
                  <Input
                    type="datetime-local"
                    value={String(field.value ?? '').slice(0, 16)}
                    onChange={(e) =>
                      field.onChange(
                        e.target.value ? `${e.target.value}:00+07:00` : null,
                      )
                    }
                  />
                </FormControl>
                <FormMessage role="alert" />
              </FormItem>
            )}
          />
        </div>
        <FormField
          control={form.control}
          name="location"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Место и адрес</FormLabel>
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
              <FormLabel>О встрече</FormLabel>
              <FormControl>
                <Textarea {...field} value={field.value ?? ''} />
              </FormControl>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="level"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Уровень участников</FormLabel>
              <Select value={field.value} onValueChange={field.onChange} items={LEVEL_ITEMS}>
                <FormControl>
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {LEVEL_ITEMS.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="requirements"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Что взять и условия участия</FormLabel>
              <FormControl>
                <Textarea {...field} value={field.value ?? ''} />
              </FormControl>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="private_info"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Точка сбора (только участникам)</FormLabel>
              <FormControl>
                <Textarea {...field} value={field.value ?? ''} />
              </FormControl>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <div className="grid gap-6 sm:grid-cols-2">
          <FormField
            control={form.control}
            name="price_note"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Стоимость / бесплатно</FormLabel>
                <FormControl>
                  <Input {...field} value={field.value ?? ''} />
                </FormControl>
                <FormMessage role="alert" />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="capacity"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Количество мест (пусто — без лимита)</FormLabel>
                <FormControl>
                  <Input
                    type="number"
                    value={field.value ?? ''}
                    onChange={(e) =>
                      field.onChange(
                        e.target.value === '' ? null : Number(e.target.value),
                      )
                    }
                  />
                </FormControl>
                <FormMessage role="alert" />
              </FormItem>
            )}
          />
        </div>
        <FormField
          control={form.control}
          name="outdoor"
          render={({ field }) => (
            <FormItem className="flex min-h-11 items-center gap-3">
              <FormControl>
                <Checkbox
                  checked={field.value ?? false}
                  onCheckedChange={field.onChange}
                />
              </FormControl>
              <FormLabel>На улице</FormLabel>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        {errors.root && (
          <Notice role="alert" tone="danger">
            {errors.root.message}
          </Notice>
        )}
        {isSubmitSuccessful && !errors.root && (
          <Notice role="status" tone="success">
            Изменения сохранены.
          </Notice>
        )}
        <Button type="submit" size="lg" disabled={isSubmitting}>
          {isSubmitting ? 'Сохраняем…' : 'Сохранить изменения'}
        </Button>
      </form>
    </Form>
  );
};
