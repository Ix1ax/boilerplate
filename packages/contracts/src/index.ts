import { z } from 'zod';

import type { components } from './api.gen';
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
// Типы ответов API генерируются из openapi.yaml (npm run gen -w @starter/contracts): вручную их не правим.
type Schemas = components['schemas'];
export type Task = Schemas['Task'];
export type TaskStatus = Task['status'];
export type TaskModeration = NonNullable<Task['moderation']>;
export type TaskPage = Schemas['TaskPage'];
export type TaskTransition = Schemas['TaskTransition'];
export type TaskHistoryPage = Schemas['TaskHistoryPage'];
export type User = Schemas['User'];
export type UploadedFile = Schemas['File'];
export type FilePage = Schemas['FilePage'];
export type ApiErrorBody = Schemas['Error'];
export type Notification = Schemas['Notification'];
export type NotificationPage = Schemas['NotificationPage'];
export type MaxStatus = Schemas['MaxStatus'];
export type TaskStats = Schemas['TaskStats'];
export type AuditRecord = Schemas['AuditRecord'];
export type AuditPage = Schemas['AuditPage'];

// Домен «Вместе»: события, запись, обсуждение, организации и «Собрать компанию».
export type Sport = Schemas['Sport'];
export type Organizer = Schemas['Organizer'];
export type Event = Schemas['Event'];
export type EventPage = Schemas['EventPage'];
export type EventInput = Schemas['EventInput'];
export type EventLevel = Event['level'];
export type JoinMode = Event['join_mode'];
export type MyParticipation = Schemas['MyParticipation'];
export type ParticipationStatus = MyParticipation['status'];
export type MyEntry = Schemas['MyEntry'];
export type Person = Schemas['Person'];
export type Participant = Schemas['Participant'];
export type EventComment = Schemas['Comment'];
export type EventWeather = Schemas['EventWeather'];
export type Organization = Schemas['Organization'];
export type OrganizationPage = Schemas['OrganizationPage'];
export type OrganizationInput = Schemas['OrganizationInput'];
export type OrganizationDetail = Schemas['OrganizationDetail'];
export type Profile = Schemas['Profile'];
export type CompanyRequest = Schemas['CompanyRequest'];
export type CompanyRequestInput = Schemas['CompanyRequestInput'];
export type Demand = Schemas['Demand'];

// Запрос к LLM-шлюзу. Пределы совпадают с проверкой в Go (server/llm): длина считается в символах.
export const llmPromptSchema = z.object({
  prompt: z.string().trim().min(1, 'Введите запрос').max(8000),
  system: z.string().trim().max(2000).optional(),
  json: z.boolean().optional(),
  temperature: z.number().min(0).max(2).optional(),
});
export type LlmPrompt = z.infer<typeof llmPromptSchema>;

// Ответ ИИ-проверки задачи. Вердикт — рекомендация: решение принимает человек.
export const taskAnalysisSchema = z.object({
  task_id: z.string(),
  verdict: z.enum(['ok', 'needs_review']),
  flags: z.array(z.object({ code: z.string(), message: z.string() })),
  advice: z
    .object({
      summary: z.string(),
      risk: z.enum(['low', 'medium', 'high']),
      suggestion: z.string(),
    })
    .nullable(),
  source: z.enum(['rules', 'rules+llm']),
  decision_required: z.boolean(),
});
export type TaskAnalysis = z.infer<typeof taskAnalysisSchema>;
