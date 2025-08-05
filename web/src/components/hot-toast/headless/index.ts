/*
 * Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { toast } from '../core/toast';

export type {
  DefaultToastOptions,
  IconTheme,
  Renderable,
  Toast,
  ToasterProps,
  ToastOptions,
  ToastPosition,
  ToastType,
  ValueFunction,
  ValueOrFunction,
} from '../core/types';

export { resolveValue } from '../core/types';
export { useToaster } from '../core/use-toaster';
export { useStore as useToasterStore } from '../core/store';

export { toast };
export default toast;
