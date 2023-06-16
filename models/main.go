package models

type Namespace struct {
	Id       string `json:"id"`
	FgaStore string `json:"fgaStore"`
}

type Document struct {
	Key         string   `json:"key"`
	Ordinal     int      `json:"ordinal"`
	NamespaceId string   `json:"namespaceId"`
	Policies    []Policy `json:"policies"`
}

type Policy struct {
	Action       string `json:"action"`
	ResourceType string `json:"resourceType"`
	Rule         string `json:"rule"`
}

type EngineRequest struct {
	Principal string         `json:"principal"`
	Resource  string         `json:"resource"`
	Context   map[string]any `json:"context"`
}
