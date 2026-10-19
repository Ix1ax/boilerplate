import type { UseFormReturn } from 'react-hook-form';
import type { EventFormInput, EventFormValues } from '@starter/contracts';

import {
  Checkbox,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Textarea,
} from '@/shared/ui';

interface FormProps {
  form: UseFormReturn<EventFormInput, unknown, EventFormValues>;
  label: string;
}

interface EventInputFieldProps extends FormProps {
  name:
    | 'title'
    | 'location'
    | 'starts_at'
    | 'ends_at'
    | 'capacity'
    | 'min_participants'
    | 'quorum_deadline'
    | 'price_note'
    | 'sport_name';
  type?: string;
}

interface EventChoiceFieldProps extends FormProps {
  name: 'sport_id' | 'level' | 'join_mode';
  items: { value: string; label: string }[];
}

interface EventTextFieldProps extends FormProps {
  name: 'description' | 'requirements' | 'private_info';
}

interface EventCheckboxFieldProps extends FormProps {
  name: 'outdoor' | 'as_organization';
}

export const EventInputField = ({
  form,
  name,
  label,
  type = 'text',
}: EventInputFieldProps) => (
  <FormField
    control={form.control}
    name={name}
    render={({ field }) => (
      <FormItem>
        <FormLabel>{label}</FormLabel>
        <FormControl>
          <Input
            {...field}
            value={
              type === 'datetime-local'
                ? String(field.value ?? '').slice(0, 16)
                : (field.value ?? '')
            }
            type={type}
            onChange={(event) => {
              const value = event.target.value;
              if (type === 'number')
                field.onChange(value === '' ? null : Number(value));
              else if (type === 'datetime-local')
                field.onChange(value ? `${value}:00+07:00` : undefined);
              else
                field.onChange(
                  name === 'sport_name' ? value || undefined : value,
                );
            }}
          />
        </FormControl>
        <FormMessage role="alert" />
      </FormItem>
    )}
  />
);

export const EventChoiceField = ({
  form,
  name,
  label,
  items,
}: EventChoiceFieldProps) => (
  <FormField
    control={form.control}
    name={name}
    render={({ field }) => (
      <FormItem>
        <FormLabel>{label}</FormLabel>
        <Select
          value={field.value ?? 'custom'}
          onValueChange={(value) =>
            field.onChange(value === 'custom' ? undefined : value)
          }
          items={items}
        >
          <FormControl>
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
          </FormControl>
          <SelectContent>
            {items.map((item) => (
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
);

export const EventTextField = ({ form, name, label }: EventTextFieldProps) => (
  <FormField
    control={form.control}
    name={name}
    render={({ field }) => (
      <FormItem>
        <FormLabel>{label}</FormLabel>
        <FormControl>
          <Textarea {...field} value={field.value ?? ''} />
        </FormControl>
        <FormMessage role="alert" />
      </FormItem>
    )}
  />
);

export const EventCheckboxField = ({
  form,
  name,
  label,
}: EventCheckboxFieldProps) => (
  <FormField
    control={form.control}
    name={name}
    render={({ field }) => (
      <FormItem className="flex min-h-11 items-center gap-3">
        <FormControl>
          <Checkbox
            checked={field.value ?? false}
            onCheckedChange={field.onChange}
          />
        </FormControl>
        <FormLabel>{label}</FormLabel>
        <FormMessage role="alert" />
      </FormItem>
    )}
  />
);
