import { useEffect, ReactNode } from "react";
import { createPortal } from "react-dom";

interface PortalProps {
  children?: ReactNode;
}

export const Portal = ({children }: PortalProps) => {
  const mount = document.getElementById("portal-root");
  const el = document.createElement("div");

  useEffect(() => {
    mount?.appendChild(el);
    return () => {
      mount?.removeChild(el);
    }
  }, [el, mount]);

  return createPortal(children, el)
};