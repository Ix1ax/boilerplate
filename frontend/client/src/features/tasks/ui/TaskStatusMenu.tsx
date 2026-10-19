import { useState } from 'react';
import { ChevronDownIcon } from 'lucide-react';

import {
  STATUS_META,
  STATUS_REQUIRES_COMMENT,
  STATUS_TRANSITIONS,
  type TaskStatus,
} from '@/entities/task';

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  Textarea,
} from '@/shared/ui';

import { useUpdateTaskStatus } from '../model';

interface TaskStatusMenuProps {
  taskId: string;
  status: TaskStatus;
}

export const TaskStatusMenu = ({ taskId, status }: TaskStatusMenuProps) => {
  const updateStatus = useUpdateTaskStatus();
  const [pendingTarget, setPendingTarget] = useState<TaskStatus | null>(null);
  const [comment, setComment] = useState('');

  const allowed = STATUS_TRANSITIONS[status];
  if (allowed.length === 0) return null;

  const requestTransition = (target: TaskStatus) => {
    if (STATUS_REQUIRES_COMMENT[target]) {
      setComment('');
      setPendingTarget(target);
      return;
    }
    updateStatus.mutate({ id: taskId, status: target });
  };

  const confirmWithComment = () => {
    if (!pendingTarget || !comment.trim()) return;
    updateStatus.mutate(
      { id: taskId, status: pendingTarget, comment: comment.trim() },
      { onSuccess: () => setPendingTarget(null) },
    );
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              variant="outline"
              size="sm"
              disabled={updateStatus.isPending}
            >
              Сменить статус
              <ChevronDownIcon />
            </Button>
          }
        />
        <DropdownMenuContent>
          {allowed.map((target) => (
            <DropdownMenuItem
              key={target}
              onClick={() => requestTransition(target)}
            >
              {STATUS_META[target].label}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog
        open={pendingTarget !== null}
        onOpenChange={(open) => !open && setPendingTarget(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {pendingTarget &&
                `Перевести в «${STATUS_META[pendingTarget].label}»`}
            </DialogTitle>
            <DialogDescription>
              Комментарий обязателен и попадёт в ленту истории.
            </DialogDescription>
          </DialogHeader>
          <Textarea
            autoFocus
            value={comment}
            onChange={(event) => setComment(event.target.value)}
            placeholder="Причина…"
            maxLength={2000}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setPendingTarget(null)}>
              Отмена
            </Button>
            <Button
              disabled={!comment.trim() || updateStatus.isPending}
              onClick={confirmWithComment}
            >
              Подтвердить
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};
