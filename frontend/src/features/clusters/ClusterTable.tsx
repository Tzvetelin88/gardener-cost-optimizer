import type { ClusterSummary } from "../../types";

interface ClusterTableProps {
  clusters: ClusterSummary[];
}

export function ClusterTable({ clusters }: ClusterTableProps) {
  return (
    <section className="panel">
      <div className="panel-header">
        <h2>Cluster inventory</h2>
        <span>{clusters.length} clusters</span>
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Cluster</th>
              <th>Cloud</th>
              <th>Seed</th>
              <th>Purpose</th>
              <th>Utilization</th>
              <th>Monthly cost</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {clusters.map((cluster) => (
              <tr key={cluster.id}>
                <td>
                  <strong>{cluster.name}</strong>
                  <div className="muted">
                    {cluster.project} / {cluster.region}
                  </div>
                </td>
                <td>{cluster.cloud}</td>
                <td>{cluster.seed}</td>
                <td>{cluster.purpose}</td>
                <td>{cluster.utilizationScore}%</td>
                <td>${cluster.monthlyCost.toFixed(0)}</td>
                <td>
                  <span className={cluster.hibernated ? "pill idle" : "pill active"}>
                    {cluster.hibernated ? "hibernated" : "running"}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
