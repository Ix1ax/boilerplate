import { z } from 'zod';

export const loginSchema = z.object({
  email: z.string().min(1, 'Укажите email').email('Неверный email'),
  password: z.string().min(1, 'Укажите пароль'),
});

export type LoginValues = z.infer<typeof loginSchema>;
