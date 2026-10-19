export const COMPANY_KEYS = {
  ALL: ['company-requests'] as const,
  MY: ['company-requests', 'my'] as const,
  MATCHES: (id: string, offset: number) =>
    ['company-requests', id, 'matches', offset] as const,
};
