import { Tooltip } from "react-tooltip";
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
    <svg className="ml-1 w-4 h-4" viewBox="0 0 72 72"><path d="M32 2C15.432 2 2 15.432 2 32s13.432 30 30 30s30-13.432 30-30S48.568 2 32 2m5 49.75H27v-24h10v24m-5-29.5a5 5 0 1 1 0-10a5 5 0 0 1 0 10" fill="currentcolor"/></svg>
    <Tooltip style= {{ maxWidth: "350px", fontSize: "12px", textTransform: "none", fontWeight: "normal", borderRadius: "0.375rem", backgroundColor: "#34343A", color: "#fff", opacity: "1" }} delayShow={100} delayHide={150} place="top" anchorId={anchorId} data-html={true} clickable={clickable}>
      {children}
    </Tooltip>
  </div>
);