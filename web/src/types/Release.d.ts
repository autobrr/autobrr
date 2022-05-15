interface Release {
  id: number;
  filter_status: string;
  rejections: string[];
  indexer: string;
  filter: string;
  protocol: string;
  title: string;
  size: number;
  raw: string;
  timestamp: Date
  action_status: ReleaseActionStatus[]
}

interface ReleaseActionStatus {
  id: number;
  status: string;
  action: string;
  type: string;
  rejections: string[];
  timestamp: string
}

interface ReleaseFindResponse {
  data: Release[];
  next_cursor: number;
  count: number;
}

interface ReleaseStats {
  total_count: number;
  filtered_count: number;
  filter_rejected_count: number;
  push_approved_count: number;
  push_rejected_count: number;
}

interface ReleaseFilter {
  id: string;
  value: string;
}
