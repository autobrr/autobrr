import { Tooltip } from "react-tooltip";
import { InformationCircleIcon } from "@heroicons/react/24/solid";
import "./CustomTooltip.css";


interface CustomTooltipProps {
    anchorId: string;
    children: React.ReactNode;
    clickable?: boolean;
    place?: string;
  }

export const CustomTooltip = ({
  anchorId,
  children,
  clickable
}: CustomTooltipProps) => (
  <div>
    <InformationCircleIcon className="ml-1 h-4 w-4 text-gray-300" aria-hidden="true" />
    <Tooltip style= {{ maxWidth: "350px", fontSize: "12px", textTransform: "none", fontWeight: "normal", borderRadius: "0.375rem", backgroundColor: "#34343A", color: "#fff", opacity: "1" }} delayShow={100} delayHide={150} place="top" anchorId={anchorId} data-html={true} clickable={clickable}>
      {children}
    </Tooltip>
  </div>
);