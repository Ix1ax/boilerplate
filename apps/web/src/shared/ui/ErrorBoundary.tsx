import { Component, type ErrorInfo, type ReactNode } from 'react';

import { logger } from '@/shared/lib';

const log = logger.child('error-boundary');

interface ErrorBoundaryProps {
  fallback?: ReactNode;
  children?: ReactNode;
}

type ErrorBoundaryState = {
  hasError: boolean;
};

/**
 * Последний рубеж для ошибок рендера вне роутера (внутри маршрутов их ловит
 * route `errorElement`). Логирует и показывает fallback.
 */
export class ErrorBoundary extends Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  state: ErrorBoundaryState = { hasError: false };

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    log.error('Непойманная ошибка рендера', error, {
      componentStack: info.componentStack,
    });
  }

  render() {
    if (!this.state.hasError) return this.props.children;

    return (
      this.props.fallback ?? (
        <div className="flex min-h-screen flex-col items-center justify-center gap-2 p-6 text-center">
          <h1 className="text-2xl font-semibold">Что-то пошло не так</h1>
          <p className="text-sm text-muted-foreground">
            Попробуйте перезагрузить страницу.
          </p>
        </div>
      )
    );
  }
}
