import { useEffect } from 'react';
import Lenis from 'lenis';
import { useAccessibility } from 'open-accessibility-widget/react';

import { useMediaQuery } from './useMediaQuery';

import 'lenis/dist/lenis.css';

/** Плавная прокрутка на время жизни страницы с учётом настроек доступности. */
export const useSmoothScroll = () => {
  const reducedMotion = useMediaQuery('(prefers-reduced-motion: reduce)');
  const { settings } = useAccessibility();
  const disabled = reducedMotion || settings.stopAnimations;

  useEffect(() => {
    if (disabled) return;

    const lenis = new Lenis({
      autoRaf: true,
      lerp: 0.12,
      smoothWheel: true,
      syncTouch: false,
      anchors: true,
      allowNestedScroll: true,
      autoToggle: true,
      stopInertiaOnNavigate: true,
    });

    return () => lenis.destroy();
  }, [disabled]);
};
