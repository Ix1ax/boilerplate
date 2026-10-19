import { z } from 'zod';

export const eventFormatInputSchema = z.object({
  format: z.enum(['offline', 'online']),
}).strict();
export const eventFormatSchema = eventFormatInputSchema.extend({
  event_id: z.string().uuid(),
});
export const videoTokenSchema = z.object({
  url: z.string().url(),
  room: z.string().min(1),
  token: z.string().min(1),
  expires_at: z.string().datetime({ offset: true }),
});
export type EventFormatInput = z.infer<typeof eventFormatInputSchema>;
export type EventFormat = z.infer<typeof eventFormatSchema>;
export type VideoToken = z.infer<typeof videoTokenSchema>;
