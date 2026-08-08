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
  vod_offset_seconds?: number;
  preview_url?: string;
  vod_url?: string;
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
  vod: string,
  sort: string,
  dir: string,
): Promise<MomentResponse> {
  const p = new URLSearchParams();
  if (at) p.set("at", at);
  if (q) p.set("q", q);
  if (survivor) p.set("survivor", survivor);
  if (language) p.set("language", language);
  if (vod && vod !== "all") p.set("vod", vod);
  if (sort) p.set("sort", sort);
  if (dir) p.set("dir", dir);
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
  vod_url?: string;
}

export function fetchStreamer(id: string): Promise<{ streamer: StreamerSummary; sessions: StreamerSession[] }> {
  return getJson(`/api/streamers/${encodeURIComponent(id)}`);
}
