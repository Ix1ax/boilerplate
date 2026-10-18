import { FileIcon } from 'lucide-react';

import { useFileList } from '@/entities/file';

import { useOffsetPagination } from '@/shared/hooks';
import { Button, EmptyState, Pagination, Skeleton } from '@/shared/ui';

import { FileItem } from './FileItem';

export const FileList = () => {
  const pagination = useOffsetPagination();
  const { data, isPending, isError, refetch } = useFileList({
    offset: pagination.offset,
    limit: pagination.pageSize,
  });

  if (isPending) return <Skeleton className="h-24 w-full" />;

  if (isError) {
    return (
      <div role="alert">
        Не удалось загрузить файлы.{' '}
        <Button onClick={() => void refetch()}>Повторить</Button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {data.items.length ? (
        <ul className="divide-y">
          {data.items.map((file) => (
            <FileItem key={file.id} file={file} />
          ))}
        </ul>
      ) : (
        <EmptyState icon={FileIcon} title="Файлов пока нет на этой странице" />
      )}
      <Pagination
        label="Страницы файлов"
        total={data.total}
        hasPrev={pagination.hasPrev}
        hasNext={pagination.hasNext(data.total)}
        onPrev={pagination.toPrev}
        onNext={pagination.toNext}
      />
    </div>
  );
};
