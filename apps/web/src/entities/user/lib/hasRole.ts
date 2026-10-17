import type { User, UserRole } from '../model';

export const hasRole = (user: User | null, allowed: readonly UserRole[]) =>
  !!user && allowed.includes(user.role);
