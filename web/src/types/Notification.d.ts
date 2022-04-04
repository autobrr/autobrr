type NotificationType = 'DISCORD';

interface Notification {
    id: number;
    name: string;
    enabled: boolean;
    type: NotificationType;
    events: string[];
    webhook: string;
}