import type { ActionRecord } from "../../types";

interface ActionPanelProps {
  actions: ActionRecord[];
}

export function ActionPanel({ actions }: ActionPanelProps) {
  return (
    <section className="panel">
      <div className="panel-header">
        <h2>Action history</h2>
        <span>{actions.length} records</span>
      </div>
      <div className="action-list">
        {actions.map((action) => (
          <article key={action.id} className="action-card">
            <div className="recommendation-topline">
              <strong>{action.target}</strong>
              <span className={`pill ${action.status === "completed" ? "active" : "idle"}`}>
                {action.status}
              </span>
            </div>
            <div className="muted">{action.type}</div>
            <p>{action.message}</p>
            <small>{new Date(action.createdAt).toLocaleString()}</small>
          </article>
        ))}
      </div>
    </section>
  );
}
