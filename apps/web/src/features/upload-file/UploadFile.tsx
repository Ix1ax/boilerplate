'use client';
import { useState } from 'react';
import type { UploadedFile } from '@starter/contracts';
import { Button, Input } from '@starter/ui';
import { api } from '@/shared/api/client';
export const UploadFile = () => {
  const [file, setFile] = useState<UploadedFile>();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  return (
    <form
      className="space-y-3"
      onSubmit={async (event) => {
        event.preventDefault();
        const form = event.currentTarget;
        setBusy(true);
        setError('');
        try {
          setFile(
            await api<UploadedFile>('/files', {
              method: 'POST',
              body: new FormData(form),
            }),
          );
          form.reset();
        } catch (error) {
          setError(error instanceof Error ? error.message : 'Ошибка загрузки');
        } finally {
          setBusy(false);
        }
      }}
    >
      <label htmlFor="file" className="text-sm font-medium">
        Файл до 10 МиБ
      </label>
      <Input
        id="file"
        type="file"
        name="file"
        required
        accept="image/png,image/jpeg,image/webp,application/pdf,text/plain"
      />
      <Button variant="outline" disabled={busy} type="submit">
        {busy ? 'Загружаем…' : 'Загрузить в S3'}
      </Button>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
      {file && (
        <p className="text-sm">
          <a className="underline" href={`/api/v1/files/${file.id}`}>
            Скачать {file.name}
          </a>{' '}
          · {file.size} байт
        </p>
      )}
    </form>
  );
};
