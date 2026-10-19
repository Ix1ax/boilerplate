import type {
  AuditEntry,
  AuditEntryDto,
  AuditPage,
  AuditPageDto,
} from './types';

export const toAuditEntry = (dto: AuditEntryDto): AuditEntry => ({
  actor: dto.actor,
  actorEmail: dto.actor_email,
  method: dto.method,
  route: dto.route,
  path: dto.path,
  status: dto.status,
  requestId: dto.request_id,
  ip: dto.ip,
  occurredAt: dto.occurred_at,
});

export const toAuditPage = (dto: AuditPageDto): AuditPage => ({
  items: dto.items.map(toAuditEntry),
  total: dto.total,
  limit: dto.limit,
  offset: dto.offset,
});
