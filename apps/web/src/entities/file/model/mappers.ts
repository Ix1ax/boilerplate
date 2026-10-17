import type { FileDto, FileMeta } from './types';

export const toFileMeta = (dto: FileDto): FileMeta => ({
  id: dto.id,
  name: dto.name,
  size: dto.size,
  contentType: dto.content_type,
  createdAt: dto.created_at,
});
