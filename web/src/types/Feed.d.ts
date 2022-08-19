interface Feed {
  id: number;
  indexer: string;
  name: string;
  type: FeedType;
  enabled: boolean;
  url: string;
  interval: number;
  api_key: string;
  created_at: Date;
  updated_at: Date;
}

type FeedType = "TORZNAB" | "RSS";

interface FeedCreate {
  indexer: string;
  name: string;
  type: FeedType;
  enabled: boolean;
  url: string;
  interval: number;
  api_key?: string;
  indexer_id: number;
}
