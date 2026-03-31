import { useEffect, useMemo, useState } from "react";
import { ActionPanel } from "../features/actions/ActionPanel";
import { ClusterTable } from "../features/clusters/ClusterTable";
import { RecommendationList } from "../features/recommendations/RecommendationList";
import { SummaryCards } from "../features/summary/SummaryCards";
import { executeAction, getActions, getClusters, getRecommendations, getSavingsSummary } from "../services/api";
import type { ActionRecord, ClusterSummary, Recommendation, SavingsSummary } from "../types";

interface LoadState {
  clusters: ClusterSummary[];
  recommendations: Recommendation[];
  actions: ActionRecord[];
  summary: SavingsSummary | null;
}

const emptyState: LoadState = {
  clusters: [],
  recommendations: [],
  actions: [],
  summary: null,
};

export function App() {
  const [data, setData] = useState<LoadState>(emptyState);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setLoading(true);
    setError(null);

    try {
      const [clusters, recommendations, actions, summary] = await Promise.all([
        getClusters(),
        getRecommendations(),
        getActions(),
        getSavingsSummary(),
      ]);

      setData({ clusters, recommendations, actions, summary });
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Unknown API error");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  const actionableCount = useMemo(
    () => data.recommendations.filter((item) => item.executable).length,
    [data.recommendations],
  );

  async function runRecommendation(recommendation: Recommendation) {
    const payload = recommendation.actionType === "hibernate-cluster"
      ? { clusterName: recommendation.targetCluster }
      : {
          sourceCluster: recommendation.sourceCluster,
          targetCluster: recommendation.targetCluster,
          namespace: recommendation.targetWorkload?.split("/")[0],
          workloadName: recommendation.targetWorkload?.split("/")[1],
        };

    await executeAction(recommendation.actionType, payload);
    await load();
  }

  return (
    <main className="layout">
      <header className="hero">
        <div>
          <p className="eyebrow">Gardener + Kubernetes</p>
          <h1>Smart Cost Optimizer</h1>
          <p className="hero-copy">
            Discover cost-saving opportunities across Gardener-managed shoot clusters,
            review the evidence, and execute approved actions.
          </p>
        </div>
        <div className="hero-side card">
          <span className="label">Ready actions</span>
          <strong>{actionableCount}</strong>
          <span className="muted">Manual approval required</span>
        </div>
      </header>

      {error ? <div className="banner error">API error: {error}</div> : null}
      {loading ? <div className="banner">Loading optimizer data...</div> : null}

      <SummaryCards summary={data.summary} />

      <section className="content-grid">
        <RecommendationList
          recommendations={data.recommendations}
          onExecute={(recommendation) => {
            void runRecommendation(recommendation);
          }}
        />
        <ActionPanel actions={data.actions} />
      </section>

      <ClusterTable clusters={data.clusters} />
    </main>
  );
}
