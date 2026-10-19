import { z } from 'zod';

const boundedText = (limit: number) => z.string().trim().refine((s) => Array.from(s).length <= limit, `Максимум ${limit} символов`);
export const proposalClaimSchema = z.object({
  location: boundedText(300).refine((s) => Array.from(s).length >= 3, 'Укажите точную точку сбора'),
  lat: z.number().finite().min(-90).max(90).nullable().optional(),
  lng: z.number().finite().min(-180).max(180).nullable().optional(),
}).refine((v) => (v.lat == null) === (v.lng == null), { message: 'Укажите обе координаты', path: ['lat'] });
export const arrivalInputSchema = z.object({
  status: z.enum(['arriving', 'late', 'here']),
  eta_minutes: z.number().int().min(0).max(120).default(0),
  description: boundedText(200).default(''),
}).refine((v) => v.status === 'late' ? v.eta_minutes >= 1 : v.eta_minutes === 0, {
  message: 'Для опоздания укажите от 1 до 120 минут, для остальных статусов — 0', path: ['eta_minutes'],
});
export type ProposalClaim = z.infer<typeof proposalClaimSchema>;
export type ArrivalInput = z.infer<typeof arrivalInputSchema>;
