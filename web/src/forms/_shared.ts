/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

export interface AddFormProps {
  isOpen: boolean;
  toggle: () => void;
}

export interface UpdateFormProps<T> {
  isOpen: boolean;
  toggle: () => void;
  data: T;
}
