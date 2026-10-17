import { DownloadIcon, Trash2Icon } from 'lucide-react';

import { useConfirm } from '@/features/confirm';

import type { FileMeta } from '@/entities/file';

import { Button } from '@/shared/ui';
import { formatBytes } from '@/shared/utils';

import { useDeleteFile, useDownloadFile } from '../model';

interface FileItemProps {
  file: FileMeta;
}

export const FileItem = ({ file }: FileItemProps) => {
  const downloadFile = useDownloadFile();
  const deleteFile = useDeleteFile();
  const confirm = useConfirm();

  const handleDelete = async () => {
    const ok = await confirm({
      title: 'Удалить файл?',
      description: file.name,
      destructive: true,
    });
    if (ok) deleteFile.mutate(file.id);
  };

  return (
    <li className="flex items-center gap-3 py-2">
      <span className="flex-1 truncate text-sm" title={file.name}>
        {file.name}
      </span>
      <span className="text-xs text-muted-foreground">
        {formatBytes(file.size)}
      </span>
      <Button
        variant="ghost"
        size="icon-sm"
        aria-label="Скачать файл"
        disabled={downloadFile.isPending}
        onClick={() => downloadFile.mutate({ id: file.id, name: file.name })}
      >
        <DownloadIcon />
      </Button>
      <Button
        variant="ghost"
        size="icon-sm"
        aria-label="Удалить файл"
        disabled={deleteFile.isPending}
        onClick={handleDelete}
      >
        <Trash2Icon />
      </Button>
    </li>
  );
};
