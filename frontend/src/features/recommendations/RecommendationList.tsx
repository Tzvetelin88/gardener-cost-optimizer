import type { Recommendation } from "../../types";

interface RecommendationListProps {
  recommendations: Recommendation[];
  onExecute: (recommendation: Recommendation) => void;
}

export function RecommendationList({
  recommendations,
  onExecute,
}: RecommendationListProps) {
  return (
    <section className="panel">
      <div className="panel-header">
        <h2>Recommendations</h2>
        <span>{recommendations.length} findings</span>
      </div>
      <div className="recommendation-list">
        {recommendations.map((item) => (
          <article key={item.id} className="recommendation-card">
            <div className="recommendation-topline">
              <div>
                <strong>{item.subject}</strong>
                <div className="muted">{item.kind}</div>
              </div>
              <span className={`pill risk-${item.risk}`}>{item.risk} risk</span>
            </div>
            <p>{item.reason}</p>
            <ul className="evidence-list">
              {item.evidence.map((entry) => (
                <li key={entry}>{entry}</li>
              ))}
            </ul>
            <div className="recommendation-footer">
              <span>${item.monthlySavings.toFixed(0)} / month</span>
              <button
                type="button"
                className="primary-button"
                disabled={!item.executable}
                onClick={() => onExecute(item)}
              >
                {item.executable ? "Run action" : "Advisory only"}
              </button>
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}
