import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { FileIcon, Trash2Icon } from 'lucide-react';
import { toast } from 'sonner';

import { useDeleteFile, useUploadFile, validateFile } from '@/features/files';

import { useFileList } from '@/entities/file';
import { TASK_KEYS } from '@/entities/task';

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  EmptyState,
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Skeleton,
} from '@/shared/ui';

import {
  createTaskSchema,
  type CreateTaskValues,
  useCreateTask,
  useUpdateTaskStatus,
} from '../model';

const TASK_ENTITY_TYPE = 'task';
const STEP_LABELS = {
  1: 'название',
  2: 'вложения',
  3: 'проверка и отправка',
} as const;

interface CreateTaskWizardProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  draftId: string | null;
}

export const CreateTaskWizard = ({
  open,
  onOpenChange,
  draftId: initialDraftId,
}: CreateTaskWizardProps) => {
  const queryClient = useQueryClient();
  // Родитель передаёт key={draftId ?? 'new'}, поэтому компонент перемонтируется
  // при каждом открытии и ленивая инициализация ниже не требует эффекта.
  const [step, setStep] = useState<1 | 2 | 3>(initialDraftId ? 2 : 1);
  const [draftId, setDraftId] = useState<string | null>(initialDraftId);
  const createTask = useCreateTask();
  const updateStatus = useUpdateTaskStatus();
  const uploadFile = useUploadFile();
  const deleteFile = useDeleteFile();

  const entity = draftId ? { type: TASK_ENTITY_TYPE, id: draftId } : undefined;
  const { data: filePage, isPending: filesLoading } = useFileList({
    entity,
    enabled: !!draftId,
  });

  const form = useForm<CreateTaskValues>({
    resolver: zodResolver(createTaskSchema),
    defaultValues: { title: '' },
  });

  const close = () => {
    onOpenChange(false);
  };

  const handleCreateDraft = (values: CreateTaskValues) => {
    createTask.mutate(values.title, {
      onSuccess: (task) => {
        setDraftId(task.id);
        setStep(2);
      },
    });
  };

  const handleUpload = (file: File) => {
    const error = validateFile(file);
    if (error) {
      toast.error(error);
      return;
    }
    if (!entity) return;
    uploadFile.mutate({ file, entity });
  };

  const handleFinish = () => {
    if (!draftId) return;
    updateStatus.mutate(
      { id: draftId, status: 'todo' },
      {
        onSuccess: () => {
          toast.success('Задача создана');
          queryClient.invalidateQueries({ queryKey: TASK_KEYS.ALL });
          close();
        },
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={(next) => !next && close()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Новая задача</DialogTitle>
          <DialogDescription>
            Шаг {step} из 3 · {STEP_LABELS[step]}
          </DialogDescription>
        </DialogHeader>

        {step === 1 && (
          <Form {...form}>
            <form
              onSubmit={form.handleSubmit(handleCreateDraft)}
              className="space-y-4"
            >
              <FormField
                control={form.control}
                name="title"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Название задачи</FormLabel>
                    <FormControl>
                      <Input
                        autoFocus
                        placeholder="Что нужно сделать"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <DialogFooter>
                <Button type="submit" disabled={createTask.isPending}>
                  {createTask.isPending ? 'Сохраняем…' : 'Далее'}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        )}

        {step === 2 && draftId && (
          <div className="space-y-4">
            <div>
              <input
                id="wizard-file-input"
                type="file"
                className="hidden"
                onChange={(event) => {
                  const file = event.target.files?.[0];
                  event.target.value = '';
                  if (file) handleUpload(file);
                }}
              />
              <Button
                variant="outline"
                disabled={uploadFile.isPending}
                onClick={() =>
                  document.getElementById('wizard-file-input')?.click()
                }
              >
                {uploadFile.isPending ? 'Загружаем…' : 'Прикрепить файл'}
              </Button>
            </div>
            {filesLoading && <Skeleton className="h-16 w-full" />}
            {!filesLoading && filePage && filePage.items.length > 0 && (
              <ul className="divide-y text-sm">
                {filePage.items.map((file) => (
                  <li
                    key={file.id}
                    className="flex items-center justify-between py-2"
                  >
                    <span className="truncate" title={file.name}>
                      {file.name}
                    </span>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label="Удалить вложение"
                      onClick={() => deleteFile.mutate(file.id)}
                    >
                      <Trash2Icon />
                    </Button>
                  </li>
                ))}
              </ul>
            )}
            {!filesLoading && (!filePage || filePage.items.length === 0) && (
              <EmptyState
                icon={FileIcon}
                title="Вложений пока нет"
                description="Можно прикрепить документы к задаче или пропустить этот шаг."
              />
            )}
            <DialogFooter className="justify-between">
              <Button variant="ghost" onClick={close}>
                Сохранить черновик и закрыть
              </Button>
              <Button onClick={() => setStep(3)}>Далее</Button>
            </DialogFooter>
          </div>
        )}

        {step === 3 && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Черновик готов к отправке. Вложений: {filePage?.total ?? 0}.
            </p>
            <DialogFooter className="justify-between">
              <Button variant="ghost" onClick={close}>
                Сохранить черновик и закрыть
              </Button>
              <Button disabled={updateStatus.isPending} onClick={handleFinish}>
                {updateStatus.isPending ? 'Создаём…' : 'Создать задачу'}
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
};
