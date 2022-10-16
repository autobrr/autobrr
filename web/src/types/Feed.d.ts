interface Feed {
  id: number;
  indexer: string;
  name: string;
  type: FeedType;
  enabled: boolean;
  url: string;
  interval: number;
  timeout: number;
  max_age: number;
  api_key: string;
  last_run: string;
  last_run_data: string;
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
  timeout: number;
  api_key?: string;
  indexer_id: number;
}
