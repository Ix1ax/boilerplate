import {
  Button,
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Notice,
} from '@/shared/ui';

import { useChangePassword } from '../model';

/** Смена пароля локального аккаунта (почта и пароль) — у «Столото ID» пароль меняется в SSO. */
export const ChangePasswordForm = () => {
  const { form, submit, errors, isSubmitting, isSubmitSuccessful } =
    useChangePassword();

  return (
    <Form {...form}>
      <form
        onSubmit={(event) => void submit(event)}
        className="grid max-w-md gap-5"
        noValidate
      >
        <FormField
          control={form.control}
          name="current_password"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Текущий пароль</FormLabel>
              <FormControl>
                <Input
                  type="password"
                  autoComplete="current-password"
                  {...field}
                />
              </FormControl>
              <FormMessage role="alert" />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="new_password"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Новый пароль</FormLabel>
              <FormControl>
                <Input type="password" autoComplete="new-password" {...field} />
              </FormControl>
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
            Пароль изменён. Остальные сессии закрыты.
          </Notice>
        )}
        <Button type="submit" disabled={isSubmitting} className="self-start">
          {isSubmitting ? 'Сохраняем…' : 'Сменить пароль'}
        </Button>
      </form>
    </Form>
  );
};
