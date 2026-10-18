import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import { TASK_KEYS, type TaskStatus } from '@/entities/task';

import { updateTaskStatus } from '../api';

export const useUpdateTaskStatus = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      id,
      status,
      comment,
    }: {
      id: string;
      status: TaskStatus;
      comment?: string;
    }) => updateTaskStatus(id, status, comment),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: TASK_KEYS.ALL });
      queryClient.invalidateQueries({ queryKey: TASK_KEYS.HISTORY(id) });
    },
    onError: (error: Error) => toast.error(error.message),
  });
};
