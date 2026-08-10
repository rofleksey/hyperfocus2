export interface Snapshot {
  id: number;
  taken_at: string;
  source: string;
  stream_count: number;
}

export interface Stream {
  streamer_id: string;
  login: string;
  display_name: string;
  profile_image_url?: string;
  viewer_count: number;
  title: string;
  language?: string;
  tags: string[];
  started_at: string;
  preview_url?: string;
  thumb_url?: string;
  twitch_url?: string;
  survivor_names: string[];
  fuzzy_score?: number;
}

export interface MomentResponse {
  requested_at: string;
  has_data: boolean;
  snapshot?: Snapshot;
  streams: Stream[];
}

async function getJson<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`request failed: ${res.status} ${res.statusText}`);
  return (await res.json()) as T;
}

export function fetchMoment(
  at: string,
  q: string,
  survivor: string,
  language: string,
  sort: string,
  dir: string,
  offset = 0,
  limit = 100,
): Promise<MomentResponse> {
  const p = new URLSearchParams();
  if (at) p.set("at", at);
  if (q) p.set("q", q);
  if (survivor) p.set("survivor", survivor);
  if (language) p.set("language", language);
  if (sort) p.set("sort", sort);
  if (dir) p.set("dir", dir);
  p.set("offset", String(offset));
  p.set("limit", String(limit));
  return getJson<MomentResponse>(`/api/moments?${p.toString()}`);
}

export function fetchSnapshots(limit = 200): Promise<{ data: Snapshot[] }> {
  return getJson(`/api/snapshots?limit=${limit}`);
}

export function fetchStreamers(q: string): Promise<{ data: StreamerSummary[] }> {
  return getJson(`/api/streamers?q=${encodeURIComponent(q)}&limit=50`);
}

export interface StreamerSummary {
  twitch_user_id: string;
  login: string;
  display_name: string;
  profile_image_url?: string;
}

export interface StreamerSession {
  id: number;
  twitch_stream_id: string;
  started_at: string;
  ended_at?: string;
}

export function fetchStreamer(id: string): Promise<{ streamer: StreamerSummary; sessions: StreamerSession[] }> {
  return getJson(`/api/streamers/${encodeURIComponent(id)}`);
}

export interface SnapshotStat {
  id: number;
  taken_at: string;
  stream_count: number;
  duration_seconds: number;
  disk_usage_bytes: number;
  preview_ok: number;
  ocr_ok: number;
  total: number;
}

export function fetchStats(n = 100): Promise<{ snapshots: SnapshotStat[] }> {
  return getJson(`/api/stats?n=${n}`);
}

export function fetchSample(streamerId: string, at?: string): Promise<Stream> {
  const p = new URLSearchParams();
  p.set("streamer_id", streamerId);
  if (at) p.set("at", at);
  return getJson<Stream>(`/api/sample?${p.toString()}`);
}
