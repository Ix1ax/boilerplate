import { z } from 'zod';
export const createTaskSchema = z.object({
  title: z
    .string()
    .trim()
    .min(1, 'Введите название')
    .refine(
      (value) => Array.from(value).length <= 200,
      'Максимум 200 символов',
    ),
});
export type CreateTask = z.infer<typeof createTaskSchema>;
export type Task = {
  id: string;
  title: string;
  completed: boolean;
  created_at: string;
};
export type User = { id: string; email: string; name: string; roles: string[] };
export type TaskPage = {
  items: Task[];
  total: number;
  limit: number;
  offset: number;
};
export type ApiErrorBody = {
  error: { code: string; message: string; request_id?: string };
};
export type UploadedFile = {
  id: string;
  name: string;
  size: number;
  content_type: string;
  created_at: string;
};
export type FilePage = {
  items: UploadedFile[];
  total: number;
  limit: number;
  offset: number;
};
