import type {
  ActionRecord,
  ClusterDetail,
  ClusterSummary,
  Recommendation,
  SavingsSummary,
} from "../types";

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

async function readJson<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: {
      "Content-Type": "application/json",
    },
    ...init,
  });

  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`);
  }

  return (await response.json()) as T;
}

export function getClusters(): Promise<ClusterSummary[]> {
  return readJson<ClusterSummary[]>("/clusters");
}

export function getRecommendations(): Promise<Recommendation[]> {
  return readJson<Recommendation[]>("/recommendations");
}

export function getActions(): Promise<ActionRecord[]> {
  return readJson<ActionRecord[]>("/actions");
}

export function getSavingsSummary(): Promise<SavingsSummary> {
  return readJson<SavingsSummary>("/savings/summary");
}

export function executeAction(type: string, payload: Record<string, unknown>) {
  return readJson<ActionRecord>(`/actions/${type}`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function wakeCluster(clusterName: string) {
  return readJson<ActionRecord>("/actions/wake-cluster", {
    method: "POST",
    body: JSON.stringify({ clusterName }),
  });
}

export function getClusterDetail(clusterName: string) {
  return readJson<ClusterDetail>(`/clusters/${encodeURIComponent(clusterName)}`);
}
