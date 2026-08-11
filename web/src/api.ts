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

async function getJson<T>(url: string, signal?: AbortSignal): Promise<T> {
  const res = await fetch(url, { signal });
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
  signal?: AbortSignal,
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
  return getJson<MomentResponse>(`/api/moments?${p.toString()}`, signal);
}

export function fetchSnapshots(limit = 200, signal?: AbortSignal): Promise<{ data: Snapshot[] }> {
  return getJson(`/api/snapshots?limit=${limit}`, signal);
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

export function fetchSample(streamerId: string, at?: string, signal?: AbortSignal): Promise<Stream> {
  const p = new URLSearchParams();
  p.set("streamer_id", streamerId);
  if (at) p.set("at", at);
  return getJson<Stream>(`/api/sample?${p.toString()}`, signal);
}
