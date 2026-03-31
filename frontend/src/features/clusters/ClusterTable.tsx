import { useState } from "react";
import type { ClusterDetail, ClusterSummary } from "../../types";
import { getClusterDetail, wakeCluster } from "../../services/api";

interface ClusterTableProps {
  clusters: ClusterSummary[];
  onRefresh: () => Promise<void>;
}

export function ClusterTable({ clusters, onRefresh }: ClusterTableProps) {
  const [expanded, setExpanded] = useState<Record<string, ClusterDetail | null>>({});
  const [loadingDetail, setLoadingDetail] = useState<Record<string, boolean>>({});
  const [wakingCluster, setWakingCluster] = useState<Record<string, boolean>>({});
  const [wakeError, setWakeError] = useState<Record<string, string>>({});

  async function toggleExpand(cluster: ClusterSummary) {
    const key = cluster.name;
    if (expanded[key] !== undefined) {
      setExpanded((prev) => { const next = { ...prev }; delete next[key]; return next; });
      return;
    }

    setLoadingDetail((prev) => ({ ...prev, [key]: true }));
    try {
      const detail = await getClusterDetail(cluster.name);
      setExpanded((prev) => ({ ...prev, [key]: detail }));
    } catch {
      setExpanded((prev) => ({ ...prev, [key]: null }));
    } finally {
      setLoadingDetail((prev) => ({ ...prev, [key]: false }));
    }
  }

  async function handleWake(cluster: ClusterSummary) {
    const key = cluster.name;
    setWakingCluster((prev) => ({ ...prev, [key]: true }));
    setWakeError((prev) => { const next = { ...prev }; delete next[key]; return next; });
    try {
      await wakeCluster(cluster.id);
      await onRefresh();
    } catch (err) {
      setWakeError((prev) => ({
        ...prev,
        [key]: err instanceof Error ? err.message : "Wake failed",
      }));
    } finally {
      setWakingCluster((prev) => ({ ...prev, [key]: false }));
    }
  }

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
              <th></th>
              <th>Cluster</th>
              <th>Cloud</th>
              <th>Seed</th>
              <th>Purpose</th>
              <th>Utilization</th>
              <th>Monthly cost</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {clusters.map((cluster) => (
              <>
                <tr key={cluster.id}>
                  <td>
                    <button
                      type="button"
                      className="expand-btn"
                      title="Show workloads"
                      onClick={() => { void toggleExpand(cluster); }}
                    >
                      {loadingDetail[cluster.name] ? (
                        <span className="btn-spinner small" />
                      ) : expanded[cluster.name] !== undefined ? (
                        "▾"
                      ) : (
                        "▸"
                      )}
                    </button>
                  </td>
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
                  <td>
                    {cluster.hibernated && (
                      <>
                        <button
                          type="button"
                          className="secondary-button"
                          disabled={wakingCluster[cluster.name]}
                          onClick={() => { void handleWake(cluster); }}
                        >
                          {wakingCluster[cluster.name] ? <span className="btn-spinner small" /> : "Wake"}
                        </button>
                        {wakeError[cluster.name] && (
                          <div className="inline-error">{wakeError[cluster.name]}</div>
                        )}
                      </>
                    )}
                  </td>
                </tr>
                {expanded[cluster.name] !== undefined && (
                  <tr key={`${cluster.id}-detail`} className="detail-row">
                    <td colSpan={9}>
                      {expanded[cluster.name] === null ? (
                        <span className="muted">Failed to load workloads.</span>
                      ) : expanded[cluster.name]!.workloads.length === 0 ? (
                        <span className="muted">No workloads discovered in this cluster.</span>
                      ) : (
                        <table className="workload-table">
                          <thead>
                            <tr>
                              <th>Namespace / Name</th>
                              <th>Kind</th>
                              <th>Replicas</th>
                              <th>CPU req</th>
                              <th>Mem GiB</th>
                              <th>Stateful</th>
                              <th>Monthly cost</th>
                            </tr>
                          </thead>
                          <tbody>
                            {expanded[cluster.name]!.workloads.map((w) => (
                              <tr key={`${w.namespace}/${w.name}`}>
                                <td>
                                  <strong>{w.name}</strong>
                                  <div className="muted">{w.namespace}</div>
                                </td>
                                <td>{w.kind}</td>
                                <td>{w.replicas}</td>
                                <td>{w.cpuRequest.toFixed(2)}</td>
                                <td>{w.memoryGiB.toFixed(2)}</td>
                                <td>
                                  <span className={w.stateful ? "pill idle" : "pill active"}>
                                    {w.stateful ? "yes" : "no"}
                                  </span>
                                </td>
                                <td>${w.monthlyCost.toFixed(0)}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      )}
                    </td>
                  </tr>
                )}
              </>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
