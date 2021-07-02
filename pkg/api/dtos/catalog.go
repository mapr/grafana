package dtos

// {
//     "name": "string",
//     "components": [{
//         "name": "string",
//         "labels": [{"key": "value"}],
//         "team": "string",
//         "namespace": "string",
//         "address": "string",
//         "pods": [{
//             "name": "string",
//             "status": "green|yellow|red"
//         }],
//         "external": "boolean"
//     }]
// }

type Catalog []CatalogService

type CatalogService struct {
	Name       string             `json:"name,omitempty"`
	Components []ServiceComponent `json:"components,omitempty"`
}

type CatalogLabels map[string]string

type CatalogPodStatus int

const (
	PodStatusGreen CatalogPodStatus = iota
	PodStatusYellow
	PodStatusRed
)

type CatalogPod struct {
	Name   string           `json:"name,omitempty"`
	Status CatalogPodStatus `json:"status"`
}

type ServiceComponent struct {
	Name      string        `json:"name,omitempty"`
	Labels    CatalogLabels `json:"labels,omitempty"`
	Teams     []string      `json:"teams,omitempty"`
	Namespace string        `json:"namespace,omitempty"`
	Address   string        `json:"address,omitempty"`
	Pods      []CatalogPod  `json:"pods,omitempty"`
	External  bool          `json:"external"`
}
