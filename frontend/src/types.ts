export interface ClusterSummary {
  id: string;
  name: string;
  project: string;
  cloud: string;
  region: string;
  seed: string;
  purpose: string;
  hibernated: boolean;
  workloadCount: number;
  utilizationScore: number;
  monthlyCost: number;
}

export interface Recommendation {
  id: string;
  kind: string;
  subject: string;
  reason: string;
  evidence: string[];
  monthlySavings: number;
  risk: "low" | "medium" | "high";
  executable: boolean;
  sourceCluster?: string;
  targetCluster?: string;
  targetWorkload?: string;
  actionType: string;
}

export interface SavingsSummary {
  totalMonthlySpend: number;
  totalMonthlySavings: number;
  actionableCount: number;
  advisoryCount: number;
}

export interface ActionRecord {
  id: string;
  type: string;
  status: string;
  target: string;
  createdAt: string;
  message: string;
}

export interface WorkloadSummary {
  cluster: string;
  namespace: string;
  name: string;
  kind: string;
  replicas: number;
  cpuRequest: number;
  memoryGiB: number;
  stateful: boolean;
  monthlyCost: number;
}

export interface ClusterDetail {
  clusterName: string;
  workloads: WorkloadSummary[];
  metrics: {
    cpuUtilizationPercent: number;
    memoryUtilizationPercent: number;
    nodeCount: number;
    idleScore: number;
  };
}
