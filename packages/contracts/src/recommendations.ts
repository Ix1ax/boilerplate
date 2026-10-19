import { z } from 'zod';

export const recommendationPreferencesSchema = z.object({
  sport_ids: z.array(z.uuid()).max(30).default([]),
  level: z.enum(['any', 'beginner', 'amateur', 'pro']).default('any'),
  city: z.string().trim().max(100).default(''),
  weekdays: z.array(z.number().int().min(1).max(7)).default([]),
  day_parts: z.array(z.enum(['morning', 'day', 'evening'])).default([]),
  lat: z.number().min(-90).max(90).nullable().default(null),
  lng: z.number().min(-180).max(180).nullable().default(null),
  radius_km: z.number().int().min(1).max(100).default(20),
}).refine((p) => (p.lat === null) === (p.lng === null), {message: 'Координаты задаются парой', path: ['lat']});
export const recommendationFeedbackSchema = z.object({ action: z.enum(['view', 'interested', 'dismiss', 'reset']) });
