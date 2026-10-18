import { useCallback, useEffect, useRef, useState } from 'react';
import type { Driver, DriveStep } from 'driver.js';
import { toast } from 'sonner';

import { logger } from '@/shared/lib';

import { escapeHtml } from '../lib/escapeHtml';
import { getTourStatus, saveTourStatus } from './progress';
import type { Tour, TourStatus } from './types';

import 'driver.js/dist/driver.css';

const POPOVER_CLASS_NAME = [
  'bg-popover! text-popover-foreground! border! border-border! rounded-lg! font-sans!',
  'max-w-[min(340px,calc(100vw-24px))]!',
  '[&_.driver-popover-title]:text-lg!',
  '[&_.driver-popover-description]:leading-normal!',
  '[&_.driver-popover-progress-text]:text-muted-foreground!',
  '[&_.driver-popover-close-btn]:text-muted-foreground!',
  '[&_.driver-popover-footer_button]:bg-secondary!',
  '[&_.driver-popover-footer_button]:text-secondary-foreground!',
  '[&_.driver-popover-footer_button]:border!',
  '[&_.driver-popover-footer_button]:border-border!',
  '[&_.driver-popover-footer_button]:rounded-lg!',
  '[&_.driver-popover-footer_button]:[text-shadow:none]!',
  '[&_.driver-popover-footer_.driver-popover-next-btn]:bg-primary!',
  '[&_.driver-popover-footer_.driver-popover-next-btn]:text-primary-foreground!',
  '[&_button:focus-visible]:outline-2!',
  '[&_button:focus-visible]:outline-ring!',
  '[&_button:focus-visible]:outline-offset-2!',
].join(' ');

const DESTROY_REASON_TO_STATUS: Record<string, TourStatus | undefined> = {
  done: 'completed',
  close: 'dismissed',
  escape: 'dismissed',
  overlay: 'dismissed',
};

// Не показываем подсказки для недоступных вкладок и скрытых контролов.
const buildSteps = (tour: Tour): DriveStep[] => {
  return tour.steps.flatMap((step) => {
    const element = step.target
      ? document.querySelector<HTMLElement>(
          `[data-tour="${CSS.escape(step.target)}"]`,
        )
      : undefined;

    if (step.target && (!element || !element.getClientRects().length)) {
      return [];
    }

    return [
      {
        element: element ?? undefined,
        popover: {
          title: escapeHtml(step.title),
          description: escapeHtml(step.description),
        },
      },
    ];
  });
}

export const useTour = (
  tour: Tour,
  userId: string | undefined,
  scope: string,
  autoStart = false,
) => {
  const active = useRef<Driver | null>(null);
  const generation = useRef(0);
  const [running, setRunning] = useState(false);

  useEffect(() => {
    return () => {
      generation.current += 1;
      active.current?.destroy();
      active.current = null;
    };
  }, [scope, userId, tour.id, tour.version]);

  const start = useCallback(async () => {
    if (!userId || active.current) return;
    const request = ++generation.current;
    const module = await import('driver.js').catch((error: unknown) => {
      logger
        .child('onboarding')
        .error('Не удалось загрузить движок обучения', error);
      return null;
    });

    if (request !== generation.current) return;

    if (!module) {
      toast.error('Не удалось загрузить обучение. Попробуйте ещё раз.');
      return;
    }

    const steps = buildSteps(tour);
    if (!steps.length) return;

    const previousFocus = document.activeElement;
    const instance = module.driver({
      steps,
      animate: !window.matchMedia('(prefers-reduced-motion: reduce)').matches,
      smoothScroll: false,
      showProgress: true,
      progressText: '{{current}} из {{total}}',
      nextBtnText: 'Далее',
      prevBtnText: 'Назад',
      doneBtnText: 'Готово',
      closeBtnLabel: 'Закрыть обучение',
      popoverClass: POPOVER_CLASS_NAME,
      disableActiveInteraction: true,
      onDestroyed: (_element, _step, { reason }) => {
        const status = DESTROY_REASON_TO_STATUS[reason];
        if (status) saveTourStatus(tour, userId, status);
        active.current = null;
        setRunning(false);
        if (previousFocus instanceof HTMLElement && previousFocus.isConnected)
          previousFocus.focus();
      },
    });

    active.current = instance;
    setRunning(true);
    instance.drive();
    if (!getTourStatus(tour, userId)) saveTourStatus(tour, userId, 'started');
  }, [tour, userId]);

  useEffect(() => {
    if (!autoStart || !userId || getTourStatus(tour, userId)) return;
    const frame = requestAnimationFrame(() => void start());

    return () => cancelAnimationFrame(frame);
  }, [autoStart, scope, start, tour, userId]);

  return { start, running };
};
