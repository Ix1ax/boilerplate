import { useFileList } from '@/entities/file';

import { FileItem } from './FileItem';

export const FileList = () => {
  const { data: files } = useFileList();

  if (!files.length) {
    return <p className="text-sm text-muted-foreground">Файлов пока нет</p>;
  }

  return (
    <ul className="divide-y">
      {files.map((file) => (
        <FileItem key={file.id} file={file} />
      ))}
    </ul>
  );
};
