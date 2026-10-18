import { useMutation, useQueryClient } from '@tanstack/react-query';

import { TASK_KEYS } from '@/entities/task';

import { deleteTask } from '../api';

export const useDeleteTask = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteTask,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TASK_KEYS.ALL });
    },
  });
};
