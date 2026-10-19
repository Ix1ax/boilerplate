import { Link } from 'react-router';

import { STATIC_ROUTES } from '@/shared/constants';
import { BrandMark } from '@/shared/ui';

const LINKS = [
  { to: STATIC_ROUTES.EVENTS, label: 'Встречи' },
  { to: STATIC_ROUTES.COMPANY, label: 'Собрать компанию' },
  { to: STATIC_ROUTES.CLUB, label: 'Клубам' },
  { to: STATIC_ROUTES.ABOUT, label: 'О сервисе' },
  { to: STATIC_ROUTES.LOGIN, label: 'Демо-вход' },
];

interface FooterProps {
  /** dark — большой подвал лендинга, light — тонкая строка внутри приложения. */
  tone?: 'light' | 'dark';
}

export const Footer = ({ tone = 'light' }: FooterProps) => {
  if (tone === 'dark') {
    return (
      <footer className="bg-ink px-4 pt-14 pb-10 text-paper sm:px-8">
        <div className="mx-auto flex max-w-[1360px] flex-wrap items-start justify-between gap-8">
          <div className="flex max-w-[420px] flex-col gap-3.5">
            <BrandMark tone="dark" />
            <span className="text-[15px] text-paper/80">
              Учебный прототип по кейсу Столото. Не является действующим
              сервисом компании.
            </span>
          </div>
          <nav aria-label="Подвал" className="flex flex-wrap gap-x-6 gap-y-1">
            {LINKS.map((link) => (
              <Link
                key={link.to}
                to={link.to}
                className="flex min-h-11 items-center text-paper/85 hover:text-paper"
              >
                {link.label}
              </Link>
            ))}
          </nav>
        </div>
      </footer>
    );
  }

  return (
    <footer className="border-t border-line bg-background dark:border-border">
      <div className="mx-auto flex max-w-[1280px] flex-wrap justify-between gap-x-6 gap-y-2 px-4 py-5 text-[15px] text-ink-soft sm:px-8 dark:text-muted-foreground">
        <span className="flex min-h-11 items-center">
          «Вместе» · учебный прототип по кейсу Столото
        </span>
        <Link
          to={STATIC_ROUTES.ABOUT}
          className="flex min-h-11 items-center font-bold text-primary hover:text-blue-hover dark:text-interest"
        >
          О сервисе и ограничениях
        </Link>
      </div>
    </footer>
  );
};
