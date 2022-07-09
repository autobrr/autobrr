import { ReactNode } from "react";

export const Tooltip = ({ children, button } : {
  message?: string, children: ReactNode, button: ReactNode
}) => {
  return (
    <div className="relative flex flex-col items-center group">
      {button}
      <div className="absolute bottom-0 flex flex-col items-center hidden mb-6 group-hover:flex">
        <span className="relative z-40 p-2 text-xs leading-none text-white whitespace-no-wrap bg-gray-600 shadow-lg rounded-md">{children}</span>
        <div className="w-3 h-3 -mt-2 rotate-45 bg-gray-600"></div>
      </div>
    </div>
  );
};