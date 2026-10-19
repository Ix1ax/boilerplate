import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  changePasswordSchema,
  type ChangePasswordValues,
} from '@starter/contracts';
import { useMutation } from '@tanstack/react-query';

import { toApiError } from '@/shared/api';

import { changeLocalPassword } from '../api';

export const useChangePassword = () => {
  const form = useForm<ChangePasswordValues>({
    resolver: zodResolver(changePasswordSchema),
    defaultValues: { current_password: '', new_password: '' },
  });
  const mutation = useMutation({ mutationFn: changeLocalPassword });
  const {
    handleSubmit,
    setError,
    reset,
    formState: { errors, isSubmitting, isSubmitSuccessful },
  } = form;
  const submit = handleSubmit(async (values) => {
    try {
      await mutation.mutateAsync(values);
      reset();
    } catch (error) {
      setError('root', { message: toApiError(error).message });
    }
  });
  return { form, submit, errors, isSubmitting, isSubmitSuccessful };
};
