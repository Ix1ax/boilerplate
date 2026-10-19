import { z } from 'zod';

/** Ответы административных API; поля соответствуют JSON сервисов. */
export interface AdminAccount {
  id: string;
  email: string;
  name: string;
  status: string;
  block_reason: string;
  roles: string[];
  last_login_at: string | null;
  created_at: string;
}
export interface AdminAccountDetail extends AdminAccount {
  history: {
    action: string;
    actor: string;
    comment: string;
    created_at: string;
  }[];
}
export interface AdminSportEntry {
  id: string;
  name: string;
  is_custom: boolean;
  hidden: boolean;
  created_by: string;
  events_count: number;
  requests_count: number;
  created_at: string;
}
export interface AdminModerationComment {
  id: string;
  event_id: string;
  event_title: string;
  author_name: string;
  body: string;
  status: string;
  hidden_reason: string;
  created_at: string;
}
export interface AdminModerationLogEntry {
  entity: string;
  entity_id: string;
  title: string;
  from: string;
  to: string;
  actor: string;
  comment: string;
  at: string;
}
export interface AdminModerationDashboard {
  organizations: { pending: number; approved: number; rejected: number };
  events: {
    upcoming: number;
    unpublished: number;
    deleted: number;
    cancelled: number;
    created_week: number;
  };
  participations: { confirmed: number; pending: number; first_time: number };
  comments: { total: number; hidden: number };
  company_requests: { active: number };
  quorum: { gathering: number; reached: number; failed: number };
  series: { date: string; events: number; participations: number }[];
  top_sports: { sport: string; events: number }[];
  pending_organizations: {
    id: string;
    name: string;
    owner_name: string;
    created_at: string;
  }[];
  updated_at: string | null;
}
export interface AdminPageResult<T> {
  items: T[];
  total: number;
  limit: number;
  offset: number;
}

/** Identity ограничивает причину блокировки пятьюстами символами. */
export const accountBlockSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(1, 'Укажите причину блокировки')
    .max(500, 'Причина — не длиннее 500 символов'),
});
