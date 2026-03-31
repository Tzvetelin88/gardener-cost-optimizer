package v1alpha1

type RecommendationSpec struct {
	ID                string   `json:"id"`
	Kind              string   `json:"kind"`
	Subject           string   `json:"subject"`
	Reason            string   `json:"reason"`
	Evidence          []string `json:"evidence"`
	MonthlySavings    float64  `json:"monthlySavings"`
	Risk              string   `json:"risk"`
	Executable        bool     `json:"executable"`
	SourceCluster     string   `json:"sourceCluster,omitempty"`
	TargetCluster     string   `json:"targetCluster,omitempty"`
	TargetWorkload    string   `json:"targetWorkload,omitempty"`
	ActionType        string   `json:"actionType"`
}

type Recommendation struct {
	APIVersion string                 `json:"apiVersion,omitempty"`
	Kind       string                 `json:"kind,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Spec       RecommendationSpec     `json:"spec"`
}
