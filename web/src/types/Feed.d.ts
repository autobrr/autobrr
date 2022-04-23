interface Feed {
    id: number;
    indexer: string;
    name: string;
    type: string;
    enabled: boolean;
    url: string;
    interval: number;
    api_key: string;
    created_at: Date;
    updated_at: Date;
}
