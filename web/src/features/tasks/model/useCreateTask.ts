import { useMutation, useQueryClient } from '@tanstack/react-query';

import { TASK_KEYS } from '@/entities/task';

import { createTask } from '../api';

export const useCreateTask = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createTask,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TASK_KEYS.ALL });
    },
  });
};
