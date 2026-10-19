import { dayPartLabels, levelLabels, weekdayLabels } from '@starter/contracts';

import type { CompanyRequest } from '@/entities/company';
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Textarea,
} from '@/shared/ui';

import { useCompanyRequestForm } from '../model';

interface CompanyRequestFormProps {
  onCreated: (request: CompanyRequest) => void;
}

export const CompanyRequestForm = ({ onCreated }: CompanyRequestFormProps) => {
  const sports = useSports();
  const { form, create, submit } = useCompanyRequestForm(onCreated);
  return (
    <Form {...form}>
      <form
        onSubmit={submit}
        className="flex flex-col gap-5 self-start rounded-3xl bg-card p-6"
      >
        <h2 className="title-display text-[26px]">Новый запрос</h2>
        <FormField
          control={form.control}
          name="sport_name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Направление</FormLabel>
              <FormControl>
                <Input
                  {...field}
                  value={field.value ?? ''}
                  list="company-sports"
                  placeholder="Например, бег"
                />
              </FormControl>
              <FormMessage role="alert">
                {'sport' in form.formState.errors
                  ? 'Выберите направление или впишите своё'
                  : undefined}
              </FormMessage>
            </FormItem>
          )}
        />
        <datalist id="company-sports">
          {sports.data?.map((s) => (
            <option key={s.id} value={s.name} />
          ))}
        </datalist>
        <FormField
          control={form.control}
          name="level"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Уровень</FormLabel>
              <Select value={field.value} onValueChange={field.onChange}>
                <FormControl>
                  <SelectTrigger className="w-full">
                    <SelectValue>
                      {levelLabels[field.value ?? 'any']}
                    </SelectValue>
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {Object.entries(levelLabels).map(([value, label]) => (
                    <SelectItem key={value} value={value}>
                      {label}
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
          name="weekdays"
          render={({ field }) => (
            <FormItem>
              <fieldset>
                <legend className="mb-2 font-medium">Удобные дни</legend>
                <div className="flex flex-wrap gap-x-4">
                  {weekdayLabels.map((label, i) => (
                    <label
                      key={label}
                      className="flex min-h-11 cursor-pointer items-center gap-2"
                    >
                      <Checkbox
                        checked={(field.value ?? []).includes(i + 1)}
                        onCheckedChange={(checked) =>
                          field.onChange(
                            checked
                              ? [...(field.value ?? []), i + 1]
                              : (field.value ?? []).filter((v) => v !== i + 1),
                          )
                        }
                      />
                      {label}
                    </label>
                  ))}
                </div>
              </fieldset>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="day_parts"
          render={({ field }) => (
            <FormItem>
              <fieldset>
                <legend className="mb-2 font-medium">Время суток</legend>
                <div className="flex flex-wrap gap-x-4">
                  {Object.entries(dayPartLabels).map(([value, label]) => (
                    <label
                      key={value}
                      className="flex min-h-11 cursor-pointer items-center gap-2"
                    >
                      <Checkbox
                        checked={(field.value ?? []).some(
                          (part) => part === value,
                        )}
                        onCheckedChange={(checked) =>
                          field.onChange(
                            checked
                              ? [...(field.value ?? []), value]
                              : (field.value ?? []).filter((v) => v !== value),
                          )
                        }
                      />
                      {label}
                    </label>
                  ))}
                </div>
              </fieldset>
              <FormDescription>
                Не выбирайте дни и время, если подходят любые.
              </FormDescription>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="area"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Район</FormLabel>
              <FormControl>
                <Input {...field} value={field.value ?? ''} />
              </FormControl>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="note"
          render={({ field }) => (
            <FormItem>
              <FormLabel>О себе и пожеланиях</FormLabel>
              <FormControl>
                <Textarea {...field} value={field.value ?? ''} />
              </FormControl>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        {create.isError && (
          <Notice tone="danger" role="alert">
            {create.error.message}
          </Notice>
        )}
        {create.isSuccess && (
          <Notice tone="success" role="status">
            Запрос создан. Подходящие встречи появятся в подборке.
          </Notice>
        )}
        <Button size="lg" type="submit" disabled={create.isPending}>
          {create.isPending ? 'Сохраняем…' : 'Искать компанию'}
        </Button>
      </form>
    </Form>
  );
};
