/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useState, ReactNode } from "react";
import { createPortal } from "react-dom";

interface PortalProps {
  children?: ReactNode;
}

export const Portal = ({children }: PortalProps) => {
  const [el] = useState(() => document.createElement("div"));

  useEffect(() => {
    const mount = document.getElementById("portal-root");
    mount?.appendChild(el);
    return () => {
      mount?.removeChild(el);
    }
  }, [el]);

  return createPortal(children, el)
};
