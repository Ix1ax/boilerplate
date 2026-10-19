import {
  cloneElement,
  type ComponentProps,
  createContext,
  type ReactElement,
  use,
  useId,
} from 'react';
import {
  Controller,
  type ControllerProps,
  type FieldPath,
  type FieldValues,
  FormProvider,
  useFormContext,
  useFormState,
} from 'react-hook-form';
import { cn } from 'cn';

import { Label } from './Label';

/** Корневой провайдер формы (= RHF FormProvider). */
const Form = FormProvider;

type FormFieldContextValue = { name: string };
const FormFieldContext = createContext<FormFieldContextValue | null>(null);

const FormField = <
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
>(
  props: ControllerProps<TFieldValues, TName>,
) => {
  return (
    <FormFieldContext value={{ name: props.name }}>
      <Controller {...props} />
    </FormFieldContext>
  );
};

type FormItemContextValue = { id: string };
const FormItemContext = createContext<FormItemContextValue | null>(null);

const FormItem = ({ className, ...props }: ComponentProps<'div'>) => {
  const id = useId();
  return (
    <FormItemContext value={{ id }}>
      <div
        data-slot="form-item"
        className={cn('grid gap-1.5', className)}
        {...props}
      />
    </FormItemContext>
  );
};

const useFormField = () => {
  const fieldContext = use(FormFieldContext);
  const itemContext = use(FormItemContext);
  const { getFieldState } = useFormContext();

  if (!fieldContext)
    throw new Error('useFormField должен быть внутри <FormField>');
  if (!itemContext)
    throw new Error('useFormField должен быть внутри <FormItem>');

  const formState = useFormState({ name: fieldContext.name });
  const fieldState = getFieldState(fieldContext.name, formState);
  const { id } = itemContext;

  return {
    id,
    name: fieldContext.name,
    formItemId: `${id}-form-item`,
    formDescriptionId: `${id}-form-item-description`,
    formMessageId: `${id}-form-item-message`,
    ...fieldState,
  };
};

const FormLabel = ({ className, ...props }: ComponentProps<typeof Label>) => {
  const { error, formItemId } = useFormField();
  return (
    <Label
      data-slot="form-label"
      data-error={!!error}
      htmlFor={formItemId}
      className={cn(className)}
      {...props}
    />
  );
};

/** Оборачивает поле (Input/Select/…), прокидывая id и aria-* из контекста формы. */
const FormControl = ({ children }: { children: ReactElement }) => {
  const { error, formItemId, formDescriptionId, formMessageId } =
    useFormField();
  return cloneElement(children as ReactElement<Record<string, unknown>>, {
    id: formItemId,
    'aria-describedby': error
      ? `${formDescriptionId} ${formMessageId}`
      : formDescriptionId,
    'aria-invalid': !!error,
  });
};

const FormDescription = ({ className, ...props }: ComponentProps<'p'>) => {
  const { formDescriptionId } = useFormField();
  return (
    <p
      data-slot="form-description"
      id={formDescriptionId}
      className={cn('text-sm text-muted-foreground', className)}
      {...props}
    />
  );
};

const FormMessage = ({
  className,
  children,
  ...props
}: ComponentProps<'p'>) => {
  const { error, formMessageId } = useFormField();
  const body = error ? String(error.message ?? '') : children;
  if (!body) return null;

  return (
    <p
      data-slot="form-message"
      id={formMessageId}
      className={cn('text-[15px] font-semibold text-danger', className)}
      {...props}
    >
      {body}
    </p>
  );
};

export {
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormDescription,
  FormMessage,
  useFormField,
};
