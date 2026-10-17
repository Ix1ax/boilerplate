import { useMutation, useQueryClient } from '@tanstack/react-query';

import { TASK_KEYS } from '@/entities/task';

import { updateTask } from '../api';

export const useToggleTask = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, completed }: { id: string; completed: boolean }) =>
      updateTask(id, completed),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TASK_KEYS.ALL });
    },
  });
};
