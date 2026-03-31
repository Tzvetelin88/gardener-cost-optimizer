package v1alpha1

type OptimizationPolicySpec struct {
	IdleThresholdPercent       int  `json:"idleThresholdPercent"`
	AllowManualHibernate       bool `json:"allowManualHibernate"`
	AllowManualWorkloadMoves   bool `json:"allowManualWorkloadMoves"`
	AllowNodePoolRightsizing   bool `json:"allowNodePoolRightsizing"`
	TargetUtilizationPercent   int  `json:"targetUtilizationPercent"`
	ConsolidationFloorPercent  int  `json:"consolidationFloorPercent"`
}

type OptimizationPolicy struct {
	APIVersion string                 `json:"apiVersion,omitempty"`
	Kind       string                 `json:"kind,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Spec       OptimizationPolicySpec `json:"spec"`
}
