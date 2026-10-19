import { useMutation, useQueryClient } from '@tanstack/react-query';

import { dismissReport, REPORT_KEYS } from '@/entities/report';

export const useDismissReport = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, resolution }: { id: string; resolution: string }) =>
      dismissReport(id, resolution),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: REPORT_KEYS.ALL });
    },
  });
};
