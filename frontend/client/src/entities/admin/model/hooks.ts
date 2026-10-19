import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  type AdminListParams,
  getAdminAccount,
  getAdminAccounts,
  getAdminComments,
  getAdminDashboard,
  getAdminLog,
  getAdminSports,
  performAdminAction,
} from '../api/adminApi';

export const useAdminDashboard = () =>
  useQuery({ queryKey: ['admin', 'dashboard'], queryFn: getAdminDashboard });
export const useAdminAccounts = (params: AdminListParams) =>
  useQuery({
    queryKey: ['admin', 'accounts', params],
    queryFn: () => getAdminAccounts(params),
  });
export const useAdminAccount = (id: string) =>
  useQuery({
    queryKey: ['admin', 'account', id],
    queryFn: () => getAdminAccount(id),
    enabled: !!id,
  });
export const useAdminSports = () =>
  useQuery({ queryKey: ['admin', 'sports'], queryFn: getAdminSports });
export const useAdminComments = (params: AdminListParams) =>
  useQuery({
    queryKey: ['admin', 'comments', params],
    queryFn: () => getAdminComments(params),
  });
export const useAdminLog = (params: AdminListParams) =>
  useQuery({
    queryKey: ['admin', 'log', params],
    queryFn: () => getAdminLog(params),
  });
export const useAdminAction = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: performAdminAction,
    onSuccess: () => {
      void client.invalidateQueries();
    },
  });
};
