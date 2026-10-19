import { Link, useLocation } from 'react-router';
import { ArrowRightIcon } from 'lucide-react';

import { LoginForm } from '@/features/auth';

import { STATIC_ROUTES } from '@/shared/constants';
import { BrandMark, RadarRings } from '@/shared/ui';

type LocationState = { from?: { pathname: string } } | null;

export const LoginPage = () => {
  const location = useLocation();
  const from = (location.state as LocationState)?.from?.pathname;

  return (
    <div className="flex min-h-screen flex-wrap bg-background">
      <section className="relative flex min-h-[520px] min-w-0 flex-[1_1_480px] flex-col justify-between gap-10 overflow-hidden bg-ink px-6 pt-8 pb-12 text-paper sm:px-12 lg:min-h-screen">
        <RadarRings
          tone="gold"
          sizes={[330, 520, 720]}
          strength={0.5}
          className="top-[58%] left-1/2"
        />
        <Link
          to={STATIC_ROUTES.HOME}
          className="relative flex min-h-11 items-center self-start"
        >
          <BrandMark tone="dark" />
          <span className="sr-only"> — на главную</span>
        </Link>
        <div className="relative">
          <p className="display-caps text-[clamp(52px,6.4vw,96px)] leading-[.86] tracking-[-0.045em]">
            Найди
            <br />
            <span className="text-gold">своих</span>
          </p>
          <p className="mt-5 max-w-[440px] text-[19px] text-paper/85">
            Аккаунт нужен, чтобы смотреть встречи, записываться, задавать
            вопросы и создавать свои.
          </p>
        </div>
        <span className="relative text-[15px] text-paper/70">
          Учебный прототип по кейсу Столото
        </span>
      </section>

      <section className="flex max-w-[720px] min-w-0 flex-[1_1_520px] flex-col justify-center gap-5 px-4 py-10 sm:px-12">
        {from && (
          <div
            role="status"
            className="flex items-center gap-3 rounded-2xl bg-gold-soft px-4 py-3.5 text-base text-ink"
          >
            <ArrowRightIcon aria-hidden className="size-5 shrink-0" />
            <span>
              После входа вернём вас на страницу, с которой вы пришли.
            </span>
          </div>
        )}
        <LoginForm />
      </section>
    </div>
  );
};
