package authzen

// Entity is used for subject, resource, and action (with type, id, and properties).
type Entity struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// EvaluationRequest matches evaluation-request.schema.json.
type EvaluationRequest struct {
	Subject  Entity `json:"subject"`
	Resource Entity `json:"resource"`
	Action   struct {
		Name       string                 `json:"name"`
		Properties map[string]interface{} `json:"properties,omitempty"`
	} `json:"action"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// EvaluationResponse matches evaluation-response.schema.json.
type EvaluationResponse struct {
	Decision bool `json:"decision"`
	Context  *struct {
		ID          string                 `json:"id"`
		ReasonAdmin map[string]interface{} `json:"reason_admin,omitempty"`
		ReasonUser  map[string]interface{} `json:"reason_user,omitempty"`
	} `json:"context,omitempty"`
}
