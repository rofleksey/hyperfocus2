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

export interface SiteConfig {
  retention_hours: number;
  notify_enabled: boolean;
}

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function getJson<T>(url: string, signal?: AbortSignal): Promise<T> {
  const res = await fetch(url, { signal });
  if (!res.ok) throw await toApiError(res);
  return (await res.json()) as T;
}

async function toApiError(res: Response): Promise<ApiError> {
  let message = `request failed: ${res.status} ${res.statusText}`;
  try {
    const body = (await res.json()) as { error?: string };
    if (body?.error) message = body.error;
  } catch {
    // non-JSON error body: keep the generic message
  }
  return new ApiError(res.status, message);
}

export function fetchConfig(signal?: AbortSignal): Promise<SiteConfig> {
  return getJson<SiteConfig>("/api/config", signal);
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
  total_viewers: number;
  duration_seconds: number;
  disk_usage_bytes: number;
  preview_ok: number;
  ocr_ok: number;
  total: number;
}

export function fetchStats(n = 100, signal?: AbortSignal): Promise<{ snapshots: SnapshotStat[] }> {
  return getJson(`/api/stats?n=${n}`, signal);
}

export interface SubscriptionPoint {
  day: string;
  new: number;
  total: number;
}

export function fetchSubscriptionStats(signal?: AbortSignal): Promise<{ points: SubscriptionPoint[] }> {
  return getJson(`/api/subscriptions/stats`, signal);
}

export function fetchSample(streamerId: string, at?: string, signal?: AbortSignal): Promise<Stream> {
  const p = new URLSearchParams();
  p.set("streamer_id", streamerId);
  if (at) p.set("at", at);
  return getJson<Stream>(`/api/sample?${p.toString()}`, signal);
}

export interface SubscriptionStatus {
  status: string;
  steam_name?: string;
}

export function fetchSubscriptionStatus(twitchLogin: string, signal?: AbortSignal): Promise<SubscriptionStatus> {
  return getJson<SubscriptionStatus>(`/api/subscribe?twitch=${encodeURIComponent(twitchLogin)}`, signal);
}

export function postSubscription(twitchLogin: string, steamUrl: string, signal?: AbortSignal): Promise<SubscriptionStatus & { message?: string }> {
  return fetch("/api/subscribe", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ twitch_login: twitchLogin, steam_url: steamUrl }),
    signal,
  }).then(async (res) => {
    if (!res.ok) throw await toApiError(res);
    return (await res.json()) as SubscriptionStatus & { message?: string };
  });
}
