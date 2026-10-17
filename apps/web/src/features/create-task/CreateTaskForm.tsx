'use client';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  createTaskSchema,
  type CreateTask,
  type Task,
} from '@starter/contracts';
import { Button, Input } from '@starter/ui';
import { tasksApi } from '@/entities/task';
interface CreateTaskFormProps {
  onCreated: (task: Task) => void;
}
export const CreateTaskForm = ({ onCreated }: CreateTaskFormProps) => {
  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<CreateTask>({
    resolver: zodResolver(createTaskSchema),
    defaultValues: { title: '' },
  });
  const submit = handleSubmit(async (values) => {
    try {
      onCreated(await tasksApi.create(values));
      reset();
    } catch (error) {
      setError('root', {
        message:
          error instanceof Error ? error.message : 'Не удалось создать задачу',
      });
    }
  });
  return (
    <form onSubmit={submit} className="space-y-2">
      <label htmlFor="task-title" className="text-sm font-medium">
        Новая задача
      </label>
      <div className="flex gap-2">
        <Input
          id="task-title"
          {...register('title')}
          placeholder="Что сделаем на этом хакатоне?"
          aria-invalid={!!errors.title}
          aria-describedby="task-error"
        />
        <Button disabled={isSubmitting} type="submit">
          {isSubmitting ? 'Создаём…' : 'Добавить'}
        </Button>
      </div>
      <p id="task-error" role="alert" className="text-sm text-destructive">
        {errors.title?.message ?? errors.root?.message}
      </p>
    </form>
  );
};
