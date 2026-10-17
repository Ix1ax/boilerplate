import { useLocation } from 'react-router';

import { STATIC_ROUTES } from '@/shared/constants';

import { login } from '../api';

type LocationState = {
  from?: { pathname: string; search?: string; hash?: string };
} | null;

export const useLogin = () => {
  const location = useLocation();
  const from = (location.state as LocationState)?.from;
  return () =>
    login(
      from
        ? `${from.pathname}${from.search ?? ''}${from.hash ?? ''}`
        : STATIC_ROUTES.DASHBOARD,
    );
};
