import { ALLOWED_FILE_TYPES, MAX_FILE_SIZE } from './constants';

export const validateFile = (file: File): string | null => {
  if (file.size > MAX_FILE_SIZE) return 'Файл больше 10 МиБ';
  if (!ALLOWED_FILE_TYPES.includes(file.type as (typeof ALLOWED_FILE_TYPES)[number])) {
    return 'Допустимы только PNG, JPEG, WebP, PDF или текстовые файлы';
  }
  return null;
};
