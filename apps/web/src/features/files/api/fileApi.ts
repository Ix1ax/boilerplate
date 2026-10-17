import { type FileDto, toFileMeta } from '@/entities/file';

import { request } from '@/shared/api';

export const uploadFile = (file: File) => {
  const data = new FormData();
  data.append('file', file);

  return request<FileDto>({
    url: '/files',
    method: 'POST',
    data,
    headers: { 'Content-Type': 'multipart/form-data' },
  }).then(toFileMeta);
};

export const downloadFile = (id: string) =>
  request<Blob>({ url: `/files/${id}`, responseType: 'blob' });

export const deleteFile = (id: string) =>
  request<void>({ url: `/files/${id}`, method: 'DELETE' });
