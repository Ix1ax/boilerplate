import { logger } from './logger';

const log = logger.child('token');

// Резерв в памяти: если localStorage недоступен (приватный режим, квота,
// отключённые куки), сессия живёт хотя бы в рамках вкладки, а не теряется молча.
const memory = new Map<string, string>();

export const readLocalStorage = (key: string): string | null => {
  try {
    const value = localStorage.getItem(key);
    if (value !== null) return value;
  } catch (error) {
    log.debug('localStorage недоступен для чтения — беру из памяти', error);
  }
  return memory.get(key) ?? null;
};

export const writeLocalStorage = (key: string, value: string) => {
  memory.set(key, value);
  try {
    localStorage.setItem(key, value);
  } catch (error) {
    log.warn(
      'localStorage недоступен — сессия в памяти, после перезагрузки пропадёт',
      error,
    );
  }
};

export const removeLocalStorage = (key: string) => {
  memory.delete(key);
  try {
    localStorage.removeItem(key);
  } catch (error) {
    log.warn('Не удалось удалить ключ из localStorage', error);
  }
};
