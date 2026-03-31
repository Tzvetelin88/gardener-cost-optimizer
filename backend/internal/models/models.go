package models

import "time"

type WorkerPool struct {
	Name        string  `json:"name"`
	MachineType string  `json:"machineType"`
	Minimum     int64   `json:"minimum"`
	Maximum     int64   `json:"maximum"`
	HourlyPrice float64 `json:"hourlyPrice"`
}

type WorkloadSummary struct {
	Cluster     string  `json:"cluster"`
	Namespace   string  `json:"namespace"`
	Name        string  `json:"name"`
	Kind        string  `json:"kind"`
	Replicas    int32   `json:"replicas"`
	CPURequest  float64 `json:"cpuRequest"`
	MemoryGiB   float64 `json:"memoryGiB"`
	Stateful    bool    `json:"stateful"`
	MonthlyCost float64 `json:"monthlyCost"`
}

type ClusterSummary struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Project          string            `json:"project"`
	Cloud            string            `json:"cloud"`
	Region           string            `json:"region"`
	Seed             string            `json:"seed"`
	Purpose          string            `json:"purpose"`
	Hibernated       bool              `json:"hibernated"`
	WorkloadCount    int               `json:"workloadCount"`
	UtilizationScore int               `json:"utilizationScore"`
	MonthlyCost      float64           `json:"monthlyCost"`
	WorkerPools      []WorkerPool      `json:"workerPools,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
}

type ClusterMetrics struct {
	CPUUtilizationPercent    float64           `json:"cpuUtilizationPercent"`
	MemoryUtilizationPercent float64           `json:"memoryUtilizationPercent"`
	NodeCount                int               `json:"nodeCount"`
	IdleScore                float64           `json:"idleScore"`
	Workloads                []WorkloadSummary `json:"workloads"`
}

type Recommendation struct {
	ID             string    `json:"id"`
	Kind           string    `json:"kind"`
	Subject        string    `json:"subject"`
	Reason         string    `json:"reason"`
	Evidence       []string  `json:"evidence"`
	MonthlySavings float64   `json:"monthlySavings"`
	Risk           string    `json:"risk"`
	Executable     bool      `json:"executable"`
	SourceCluster  string    `json:"sourceCluster,omitempty"`
	TargetCluster  string    `json:"targetCluster,omitempty"`
	TargetWorkload string    `json:"targetWorkload,omitempty"`
	ActionType     string    `json:"actionType"`
	CreatedAt      time.Time `json:"createdAt"`
}

type SavingsSummary struct {
	TotalMonthlySpend   float64 `json:"totalMonthlySpend"`
	TotalMonthlySavings float64 `json:"totalMonthlySavings"`
	ActionableCount     int     `json:"actionableCount"`
	AdvisoryCount       int     `json:"advisoryCount"`
}

type ActionRecord struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status"`
	Target    string                 `json:"target"`
	Message   string                 `json:"message"`
	CreatedAt time.Time              `json:"createdAt"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type InventorySnapshot struct {
	Clusters        []ClusterSummary `json:"clusters"`
	Recommendations []Recommendation `json:"recommendations"`
	Summary         SavingsSummary   `json:"summary"`
}
