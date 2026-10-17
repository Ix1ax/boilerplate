import type { ChangeEvent } from 'react';
import { useRef } from 'react';
import { UploadIcon } from 'lucide-react';
import { toast } from 'sonner';

import { Button } from '@/shared/ui';

import { useUploadFile, validateFile } from '../model';

export const UploadFileForm = () => {
  const inputRef = useRef<HTMLInputElement>(null);
  const uploadFile = useUploadFile();

  const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;

    const error = validateFile(file);
    if (error) {
      toast.error(error);
      return;
    }

    uploadFile.mutate(file);
  };

  return (
    <div>
      <input
        ref={inputRef}
        type="file"
        className="hidden"
        onChange={handleChange}
      />
      <Button
        variant="outline"
        disabled={uploadFile.isPending}
        onClick={() => inputRef.current?.click()}
      >
        <UploadIcon />
        {uploadFile.isPending ? 'Загружаем…' : 'Загрузить файл'}
      </Button>
    </div>
  );
};
