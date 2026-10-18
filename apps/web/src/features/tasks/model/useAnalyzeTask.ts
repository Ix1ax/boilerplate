import { useMutation } from '@tanstack/react-query';

import { analyzeTask } from '../api';

/** ИИ-проверка задачи: только рекомендация, данные на сервере не меняются. */
export const useAnalyzeTask = () => useMutation({ mutationFn: analyzeTask });
