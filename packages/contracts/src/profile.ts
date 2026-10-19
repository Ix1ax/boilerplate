import { z } from 'zod';

import { organizationInputSchema } from './sport';

export const profilePreferencesSchema = z.object({
  notify_reminders: z.boolean(),
  notify_decisions: z.boolean(),
  notify_replies: z.boolean(),
  large_text: z.boolean(),
  reduced_motion: z.boolean(),
});
export const profileSettingsSchema = z.object({
  about: z.string().trim().max(1000, 'О себе — до 1000 символов'),
  city: z.string().trim().max(100, 'Город — до 100 символов'),
  level: z.enum(['', 'beginner', 'amateur', 'pro']),
  interests: z.array(z.string().trim().min(2).max(60)).max(20),
  preferences: profilePreferencesSchema,
});
export type ProfileSettingsValues = z.infer<typeof profileSettingsSchema>;

export const localAuthFormSchema = z.discriminatedUnion('mode', [
  z.object({
    mode: z.literal('login'),
    email: z.email('Проверьте email'),
    password: z.string().min(1, 'Введите пароль'),
    name: z.string().optional(),
  }),
  z.object({
    mode: z.literal('register'),
    email: z.email('Проверьте email'),
    password: z.string().min(8, 'Пароль — не короче 8 символов'),
    name: z.string().trim().min(1, 'Введите имя').max(100),
  }),
]);
export type LocalAuthFormValues = z.infer<typeof localAuthFormSchema>;

export const organizationProfileSchema = organizationInputSchema.extend({
  site: z
    .string()
    .trim()
    .max(500)
    .refine(
      (value) => !value || (/^https?:\/\//.test(value) && URL.canParse(value)),
      'Укажите адрес сайта с https:// или http://',
    )
    .default(''),
  zones: z
    .array(z.string().trim().min(1, 'Укажите район').max(150))
    .max(20)
    .default([]),
  venues: z
    .array(z.string().trim().min(1, 'Укажите площадку').max(150))
    .max(20)
    .default([]),
});
export type OrganizationProfileValues = z.output<
  typeof organizationProfileSchema
>;
export const organizationApplicationSchema = organizationProfileSchema.extend({
  representative: z
    .boolean()
    .refine((value) => value, 'Подтвердите, что представляете сообщество'),
});
