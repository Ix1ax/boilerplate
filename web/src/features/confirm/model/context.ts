import { createContext } from 'react';

import type { ConfirmOptions } from './types';

export type Confirm = (options: ConfirmOptions) => Promise<boolean>;

export const ConfirmContext = createContext<Confirm | null>(null);
