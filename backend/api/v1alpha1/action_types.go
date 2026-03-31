package v1alpha1

type HibernateClusterRequest struct {
	ClusterName string `json:"clusterName"`
}

type ScaleNodePoolRequest struct {
	ClusterName string `json:"clusterName"`
	WorkerPool  string `json:"workerPool"`
	Minimum     int64  `json:"minimum"`
	Maximum     int64  `json:"maximum"`
}

type MoveWorkloadRequest struct {
	SourceCluster string `json:"sourceCluster"`
	TargetCluster string `json:"targetCluster"`
	Namespace     string `json:"namespace"`
	WorkloadName  string `json:"workloadName"`
}

type ActionStatus struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Target    string `json:"target"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}
