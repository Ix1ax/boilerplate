import { z } from 'zod';

const text = (max: number) => z.string().trim().min(1, 'Введите текст').refine(
  value => Array.from(value).length <= max, `Не более ${max} символов`,
);
export const socialFriendActionSchema = z.enum(['request', 'accept', 'decline', 'remove']);
export const socialChatInputSchema = z.object({ user_id: z.string().trim().min(1).max(200) }).strict();
export const socialMessageInputSchema = z.object({ body: text(2000) }).strict();
export const socialReviewInputSchema = z.object({ score: z.number().int().min(1).max(5), body: text(2000) }).strict();
export const socialStoryInputSchema = z.object({ body: text(1000) }).strict();
export const socialReasonInputSchema = z.object({ reason: text(500) }).strict();
export const socialPaginationSchema = z.object({ cursor: z.string().max(512).optional(), limit: z.coerce.number().int().min(1).max(100).default(30) });
export type SocialMessageInput = z.infer<typeof socialMessageInputSchema>;
export type SocialReviewInput = z.infer<typeof socialReviewInputSchema>;
export type SocialStoryInput = z.infer<typeof socialStoryInputSchema>;
