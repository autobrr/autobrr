type NotificationType = "DISCORD" | "TELEGRAM";
type NotificationEvent = "PUSH_APPROVED" | "PUSH_REJECTED" | "PUSH_ERROR" | "IRC_DISCONNECTED" | "IRC_RECONNECTED" | "APP_UPDATE_AVAILABLE";

interface Notification {
  id: number;
  name: string;
  enabled: boolean;
  type: NotificationType;
  events: NotificationEvent[];
  webhook?: string;
  token?: string;
  channel?: string;
}