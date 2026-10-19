import type { CSSProperties } from 'react';
import { cn } from 'cn';
import { useReducedMotion } from 'motion/react';
import { useAccessibility } from 'open-accessibility-widget/react';

import { useTimeout } from '@/shared/hooks';
import { STOLOTO_PATH } from '@/shared/ui';

interface LoadingPageProps {
  onComplete: () => void;
}

const letters = [
  [27, 15],
  [42, 15.3],
  [57.3, 16.9],
  [74.2, 16.8],
  [91, 17],
  [108, 14.7],
  [122.7, 16.3],
];

const ticket = STOLOTO_PATH.slice(0, STOLOTO_PATH.indexOf('Z') + 1);

export const LoadingPage = ({ onComplete }: LoadingPageProps) => {
  const reducedMotion = useReducedMotion();
  const { settings } = useAccessibility();
  const reduce = reducedMotion || settings.stopAnimations;

  useTimeout(onComplete, reduce ? 200 : 4200);

  return (
    <div
      className={cn(
        'grid min-h-dvh w-full animate-intro-exit overflow-hidden place-items-center bg-intro-yellow text-[#121212] motion-reduce:animate-none motion-reduce:[&_*]:animate-none!',
        reduce && 'animate-none! [&_*]:animate-none!',
      )}
      onAnimationEnd={(event) => {
        if (
          event.target === event.currentTarget &&
          event.animationName === 'intro-exit'
        ) {
          onComplete();
        }
      }}
    >
      <span className="sr-only" role="status">
        Загружаем Столото
      </span>
      <svg
        className="h-auto w-[min(80vw,480px)] overflow-visible"
        viewBox="0 0 138 40"
        aria-hidden="true"
      >
        <defs>
          <mask
            id="intro-person-mask"
            maskUnits="userSpaceOnUse"
            x="0"
            y="8"
            width="23"
            height="24"
          >
            <circle cx="10.87" cy="19.43" r="10.6" fill="white" />
            <path d={STOLOTO_PATH} fill="black" fillRule="evenodd" />
          </mask>
          {letters.map(([x, width], index) => (
            <clipPath key={x} id={`intro-letter-${index}`}>
              <rect x={x} width={width} height="40" />
            </clipPath>
          ))}
        </defs>
        <g
          className={cn(
            'animate-intro-assembled motion-reduce:hidden',
            reduce && 'hidden',
          )}
        >
          <g className="animate-intro-move">
            <g className="origin-[10.87px_30.3px] animate-intro-drop">
              <circle cx="10.87" cy="19.43" r="10.87" fill="currentColor" />
              <g className="origin-[10.87px_26px] animate-intro-person">
                <circle
                  cx="10.87"
                  cy="19.43"
                  r="10.6"
                  className="fill-intro-yellow"
                  mask="url(#intro-person-mask)"
                />
              </g>
              <path
                className="origin-[14px_6px] animate-intro-ticket"
                d={ticket}
                fill="currentColor"
              />
            </g>
          </g>
          {letters.map(([x], index) => (
            <g key={x} clipPath={`url(#intro-letter-${index})`}>
              <g
                className="animate-intro-letter [animation-delay:var(--letter-delay)]"
                style={
                  {
                    '--letter-delay': `${2100 + index * 65}ms`,
                  } as CSSProperties
                }
              >
                <path d={STOLOTO_PATH} fill="currentColor" fillRule="evenodd" />
              </g>
            </g>
          ))}
        </g>
        <path
          className="animate-intro-settled"
          d={STOLOTO_PATH}
          fill="currentColor"
          fillRule="evenodd"
        />
      </svg>
    </div>
  );
};
