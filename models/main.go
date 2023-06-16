package models

type Namespace struct {
	Id       string `json:"id"`
	FgaStore string `json:"fgaStore"`
}

type Document struct {
	Key         string   `json:"key"`
	Ordinal     int      `json:"ordinal"`
	NamespaceId string   `json:"namespace"`
	Policies    []Policy `json:"policies"`
}

type Policy struct {
	Action       string `json:"action"`
	ResourceType string `json:"resourceType"`
	Rule         string `json:"rule"`
}

type GetPermissionsRequest struct {
	NamespaceId string         `json:"namespace"`
	PrincipalId string         `json:"principal"`
	ResourceId  string         `json:"resource"`
	Context     map[string]any `json:"context"`
	Policies    []Policy       `json:"policies"`
}

type EngineRequestOptions struct {
	PrincipalId string
	Principal   map[string]any

	ResourceId string
	Resource   map[string]any

	Context map[string]any
	Perms   []string
	Rel     func(string) bool
	Full    func(string, string) bool
}
