type NotificationType = "DISCORD" | "TELEGRAM";

interface Notification {
  id: number;
  name: string;
  enabled: boolean;
  type: NotificationType;
  events: string[];
  webhook?: string;
  token?: string;
  channel?: string;
}