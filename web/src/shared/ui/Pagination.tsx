import { Button } from './Button';

interface PaginationProps {
  total: number;
  hasPrev: boolean;
  hasNext: boolean;
  onPrev: () => void;
  onNext: () => void;
  label?: string;
}

/** Общий паттерн «Назад/Далее/Всего» для постраничных списков (задачи, файлы, аудит). */
export const Pagination = ({
  total,
  hasPrev,
  hasNext,
  onPrev,
  onNext,
  label = 'Страницы',
}: PaginationProps) => {
  return (
    <nav aria-label={label} className="flex items-center gap-3">
      <Button variant="outline" disabled={!hasPrev} onClick={onPrev}>
        Назад
      </Button>
      <span className="text-sm">Всего: {total}</span>
      <Button variant="outline" disabled={!hasNext} onClick={onNext}>
        Далее
      </Button>
    </nav>
  );
};
