import type { Task, TaskDto, TaskPage, TaskPageDto } from './types';

export const toTask = (dto: TaskDto): Task => ({
  id: dto.id,
  title: dto.title,
  completed: dto.completed,
  createdAt: dto.created_at,
});

export const toTaskPage = (dto: TaskPageDto): TaskPage => ({
  items: dto.items.map(toTask),
  total: dto.total,
  limit: dto.limit,
  offset: dto.offset,
});
