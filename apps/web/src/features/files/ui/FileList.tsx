import { useFileList } from '@/entities/file';

import { useOffsetPagination } from '@/shared/hooks';
import { Button, Skeleton } from '@/shared/ui';

import { FileItem } from './FileItem';

export const FileList = () => {
  const pagination = useOffsetPagination();
  const { data, isPending, isError, refetch } = useFileList(
    pagination.offset,
    pagination.pageSize,
  );

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
        <p className="text-sm text-muted-foreground">
          Файлов пока нет на этой странице
        </p>
      )}
      <nav aria-label="Страницы файлов" className="flex items-center gap-3">
        <Button
          variant="outline"
          disabled={!pagination.hasPrev}
          onClick={pagination.toPrev}
        >
          Назад
        </Button>
        <span className="text-sm">Всего: {data.total}</span>
        <Button
          variant="outline"
          disabled={!pagination.hasNext(data.total)}
          onClick={pagination.toNext}
        >
          Далее
        </Button>
      </nav>
    </div>
  );
};
