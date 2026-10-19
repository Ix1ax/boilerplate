import { useLocation, useSearchParams } from 'react-router';

import { STATIC_ROUTES } from '@/shared/constants';

import { login } from '../api';

type LocationState = {
  from?: { pathname: string; search?: string; hash?: string };
} | null;

/** Куда вернуть после входа: из state защищённого маршрута или из ?next=. */
export const useLoginTarget = (fallback: string = STATIC_ROUTES.EVENTS) => {
  const location = useLocation();
  const [search] = useSearchParams();
  const from = (location.state as LocationState)?.from;
  if (from) return `${from.pathname}${from.search ?? ''}${from.hash ?? ''}`;
  const next = search.get('next');
  // Только свои пути: чужой адрес в ?next= не должен уводить с сайта.
  if (
    next &&
    next.startsWith('/') &&
    !next.startsWith('//') &&
    !next.includes('\\') &&
    !/[\u0000-\u001f]/.test(next)
  )
    return next;
  return fallback;
};

export const useLogin = () => {
  const target = useLoginTarget();
  return (next: string = target) => login(next);
};
