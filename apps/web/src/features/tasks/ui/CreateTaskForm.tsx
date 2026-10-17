import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';

import { Button, Form, FormControl, FormField, FormItem, FormMessage, Input } from '@/shared/ui';

import { createTaskSchema, type CreateTaskValues, useCreateTask } from '../model';

export const CreateTaskForm = () => {
  const createTask = useCreateTask();
  const form = useForm<CreateTaskValues>({
    resolver: zodResolver(createTaskSchema),
    defaultValues: { title: '' },
  });

  const onSubmit = (values: CreateTaskValues) => {
    createTask.mutate(values.title, {
      onSuccess: () => form.reset(),
    });
  };

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className="flex items-start gap-2"
      >
        <FormField
          control={form.control}
          name="title"
          render={({ field }) => (
            <FormItem className="flex-1">
              <FormControl>
                <Input placeholder="Новая задача" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button type="submit" disabled={createTask.isPending}>
          {createTask.isPending ? 'Добавляем…' : 'Добавить'}
        </Button>
      </form>
    </Form>
  );
};
