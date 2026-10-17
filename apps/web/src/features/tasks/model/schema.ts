import { z } from 'zod';

export const createTaskSchema = z.object({
  title: z.string().trim().min(1, 'Укажите название').max(200, 'Слишком длинное название'),
});

export type CreateTaskValues = z.infer<typeof createTaskSchema>;
