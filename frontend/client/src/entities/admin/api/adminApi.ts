import type {
  AdminAccount,
  AdminAccountDetail,
  AdminModerationComment,
  AdminModerationDashboard,
  AdminModerationLogEntry,
  AdminPageResult,
  AdminSportEntry,
} from '@starter/contracts';

import { type Camelize, request } from '@/shared/api';

export type Account = Camelize<AdminAccount>;
export type AccountDetail = Camelize<AdminAccountDetail>;
export type AdminSport = Camelize<AdminSportEntry>;
export type AdminComment = Camelize<AdminModerationComment>;
export type AdminLog = Camelize<AdminModerationLogEntry>;
export type Dashboard = Camelize<AdminModerationDashboard>;
export interface AdminListParams {
  q?: string;
  status?: string;
  offset?: number;
  entity?: string;
  event_id?: string;
}
export const getAdminDashboard = () =>
  request<Dashboard>({ url: '/admin/moderation/dashboard' });
export const getAdminAccounts = (params: AdminListParams) =>
  request<AdminPageResult<Account>>({
    url: '/admin/accounts',
    params: { limit: 20, ...params },
  });
export const getAdminAccount = (id: string) =>
  request<AccountDetail>({ url: `/admin/accounts/${id}` });
export const getAdminSports = () =>
  request<{ items: AdminSport[] }>({ url: '/admin/sports' });
export const getAdminComments = (params: AdminListParams) =>
  request<AdminPageResult<AdminComment>>({
    url: '/admin/moderation/comments',
    params: { limit: 20, ...params },
  });
export const getAdminLog = (params: AdminListParams) =>
  request<AdminPageResult<AdminLog>>({
    url: '/admin/moderation/log',
    params: { limit: 20, ...params },
  });
export interface AdminAction {
  url: string;
  reason?: string;
}
export const performAdminAction = ({ url, reason }: AdminAction) =>
  request<unknown>({
    url,
    method: 'POST',
    data: reason === undefined ? undefined : { reason },
  });
