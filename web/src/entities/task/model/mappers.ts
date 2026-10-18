import type {
  Task,
  TaskDto,
  TaskHistoryPageDto,
  TaskPage,
  TaskPageDto,
  TaskStatus,
  TaskTransition,
  TaskTransitionDto,
} from './types';

export const toTask = (dto: TaskDto): Task => ({
  id: dto.id,
  title: dto.title,
  status: dto.status as TaskStatus,
  createdAt: dto.created_at,
  moderation: dto.moderation
    ? { verdict: dto.moderation.verdict, source: dto.moderation.source }
    : null,
});

export const toTaskPage = (dto: TaskPageDto): TaskPage => ({
  items: dto.items.map(toTask),
  total: dto.total,
  limit: dto.limit,
  offset: dto.offset,
});

export const toTransition = (dto: TaskTransitionDto): TaskTransition => ({
  from: dto.from as TaskStatus,
  to: dto.to as TaskStatus,
  actor: dto.actor,
  comment: dto.comment,
  createdAt: dto.created_at,
});

export const toTransitions = (dto: TaskHistoryPageDto): TaskTransition[] =>
  dto.items.map(toTransition);
