export type AuditEntry = {
  actor: string;
  actorEmail: string;
  method: string;
  route: string;
  path: string;
  status: number;
  requestId: string;
  ip: string;
  occurredAt: string;
};

export type AuditPage = {
  items: AuditEntry[];
  total: number;
  limit: number;
  offset: number;
};

export type AuditListParams = {
  limit?: number;
  offset?: number;
  q?: string;
  actor?: string;
  method?: string;
  status?: string;
  requestId?: string;
  sort?: string;
};

// DTO с проволоки — форма ответа Go API из общего контракта, не доменная модель выше.
export type {
  AuditPage as AuditPageDto,
  AuditRecord as AuditEntryDto,
} from '@starter/contracts';
