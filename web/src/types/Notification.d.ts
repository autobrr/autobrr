type NotificationType = "DISCORD" | "NOTIFIARR" | "TELEGRAM" | "PUSHOVER" | "WEBHOOK";
type NotificationEvent =
  "PUSH_APPROVED"
  | "PUSH_REJECTED"
  | "PUSH_ERROR"
  | "IRC_DISCONNECTED"
  | "IRC_RECONNECTED"
  | "APP_UPDATE_AVAILABLE";

type NotificationWebhookMethodType = "POST" | "PUT" | "GET";

type NotificationWebhookDataType = "application/json" | "application/x-www-form-urlencoded";

interface Notification {
  id: number;
  name: string;
  enabled: boolean;
  type: NotificationType;
  events: NotificationEvent[];
  webhook?: string;
  token?: string;
  api_key?: string;
  channel?: string;
  priority?: number;
}
