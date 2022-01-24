interface Indexer {
    id: number;
    name: string;
    identifier: string;
    enabled: boolean;
    settings: object | any;
}

interface IndexerSchema {
    name: string;
    identifier: string;
    description: string;
    language: string;
    privacy: string;
    protocol: string;
    urls: string[];
    settings: IndexerSchemaSettings[];
    irc: IndexerSchemaIRC;
}

interface IndexerSchemaSettings {
    name: string;
    type: string;
    required: boolean;
    label: string;
    help: string;
    description: string;
    default: string;
}

interface IndexerSchemaIRC {
    network: string;
    server: string;
    port: number;
    tls: boolean;
    nickserv: boolean;
    announcers: string[];
    channels: string[];
    invite: string[];
    invite_command: string;
    settings: IndexerSchemaSettings[];
}