import { z } from 'zod';
import { companyRequestSchema, type CompanyRequest } from './sport';

export const companyWindowSchema = z
  .object({
    starts_at: z.string().datetime({ offset: true }),
    ends_at: z.string().datetime({ offset: true }),
  })
  .refine(
    (v) =>
      new Date(v.ends_at).getTime() - new Date(v.starts_at).getTime() >=
      3_600_000,
    { message: 'Окно должно длиться хотя бы час' },
  );

export const companyPlanSchema = companyRequestSchema
  .safeExtend({
    zones: z
      .array(z.string().trim().min(1).max(100))
      .min(1, 'Выберите хотя бы одну зону')
      .max(8),
    windows: z
      .array(companyWindowSchema)
      .min(1, 'Добавьте хотя бы одно окно')
      .max(20),
    expires_at: z.string().datetime({ offset: true }),
  })
  .refine(
    (v) =>
      new Date(v.expires_at) > new Date() &&
      new Date(v.expires_at).getTime() <= Date.now() + 30 * 86_400_000,
    {
      path: ['expires_at'],
      message: 'Срок запроса — в пределах следующих 30 дней',
    },
  )
  .refine(
    (v) =>
      v.windows.every(
        (w) =>
          new Date(w.starts_at) > new Date() &&
          new Date(w.ends_at) <= new Date(v.expires_at),
      ),
    {
      path: ['windows'],
      message: 'Окна должны быть в будущем и заканчиваться до срока запроса',
    },
  );

export type CompanyPlanInput = z.input<typeof companyPlanSchema>;
export type CompanyPlanValues = z.output<typeof companyPlanSchema>;
export type CompanyPlan = CompanyRequest & {
  zones: string[];
  windows: z.infer<typeof companyWindowSchema>[];
  revision: number;
};
export type CompanyProposal = {
  version: string;
  sport_id: string;
  sport_name: string;
  zone: string;
  level: 'any' | 'beginner' | 'amateur' | 'pro';
  starts_at: string;
  ends_at: string;
  count: number;
  interested: number;
  response: boolean | null;
  event_id: string | null;
  enrolled: number;
};

export const companyResponseSchema = z.object({
  version: z.string().min(1).max(100),
  interested: z.boolean(),
});
export type CompanyResponseValues = z.infer<typeof companyResponseSchema>;
