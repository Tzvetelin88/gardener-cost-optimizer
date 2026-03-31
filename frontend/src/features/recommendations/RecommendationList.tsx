import { useState } from "react";
import type { Recommendation } from "../../types";

interface RecommendationListProps {
  recommendations: Recommendation[];
  onExecute: (recommendation: Recommendation) => Promise<void>;
}

const KIND_LABELS: Record<string, string> = {
  "idle-cluster": "Idle cluster",
  "cheaper-placement": "Cheaper placement",
  "cluster-consolidation": "Consolidation",
};

const RISK_OPTIONS = ["low", "medium", "high"] as const;

export function RecommendationList({
  recommendations,
  onExecute,
}: RecommendationListProps) {
  const [kindFilter, setKindFilter] = useState<string | null>(null);
  const [riskFilter, setRiskFilter] = useState<string | null>(null);
  const [running, setRunning] = useState<Record<string, boolean>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});

  const kinds = [...new Set(recommendations.map((r) => r.kind))];

  const filtered = recommendations.filter((r) => {
    if (kindFilter !== null && r.kind !== kindFilter) return false;
    if (riskFilter !== null && r.risk !== riskFilter) return false;
    return true;
  });

  async function handleExecute(item: Recommendation) {
    setRunning((prev) => ({ ...prev, [item.id]: true }));
    setErrors((prev) => { const next = { ...prev }; delete next[item.id]; return next; });
    try {
      await onExecute(item);
    } catch (err) {
      setErrors((prev) => ({
        ...prev,
        [item.id]: err instanceof Error ? err.message : "Action failed",
      }));
    } finally {
      setRunning((prev) => ({ ...prev, [item.id]: false }));
    }
  }

  return (
    <section className="panel">
      <div className="panel-header">
        <h2>Recommendations</h2>
        <span>{filtered.length} / {recommendations.length} findings</span>
      </div>

      <div className="filter-bar">
        <div className="filter-group">
          <button
            type="button"
            className={`filter-pill ${kindFilter === null ? "active" : ""}`}
            onClick={() => setKindFilter(null)}
          >
            All kinds
          </button>
          {kinds.map((kind) => (
            <button
              key={kind}
              type="button"
              className={`filter-pill ${kindFilter === kind ? "active" : ""}`}
              onClick={() => setKindFilter(kindFilter === kind ? null : kind)}
            >
              {KIND_LABELS[kind] ?? kind}
            </button>
          ))}
        </div>
        <div className="filter-group">
          <button
            type="button"
            className={`filter-pill ${riskFilter === null ? "active" : ""}`}
            onClick={() => setRiskFilter(null)}
          >
            All risks
          </button>
          {RISK_OPTIONS.map((risk) => (
            <button
              key={risk}
              type="button"
              className={`filter-pill risk-${risk} ${riskFilter === risk ? "active" : ""}`}
              onClick={() => setRiskFilter(riskFilter === risk ? null : risk)}
            >
              {risk}
            </button>
          ))}
        </div>
      </div>

      <div className="recommendation-list">
        {filtered.length === 0 && (
          <p className="muted" style={{ marginTop: 16 }}>No recommendations match the current filters.</p>
        )}
        {filtered.map((item) => (
          <article key={item.id} className="recommendation-card">
            <div className="recommendation-topline">
              <div>
                <strong>{item.subject}</strong>
                <div className="muted">{KIND_LABELS[item.kind] ?? item.kind}</div>
              </div>
              <span className={`pill risk-${item.risk}`}>{item.risk} risk</span>
            </div>
            <p>{item.reason}</p>
            <ul className="evidence-list">
              {item.evidence.map((entry) => (
                <li key={entry}>{entry}</li>
              ))}
            </ul>
            {errors[item.id] && (
              <div className="inline-error">{errors[item.id]}</div>
            )}
            <div className="recommendation-footer">
              <span>${item.monthlySavings.toFixed(0)} / month</span>
              <button
                type="button"
                className="primary-button"
                disabled={!item.executable || running[item.id]}
                onClick={() => { void handleExecute(item); }}
              >
                {running[item.id] ? (
                  <span className="btn-spinner" />
                ) : item.executable ? (
                  "Run action"
                ) : (
                  "Advisory only"
                )}
              </button>
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}
