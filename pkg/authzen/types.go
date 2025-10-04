// Package authzen provides types and functions for the AuthZEN protocol.
// AuthZEN is an authorization protocol that allows for policy decisions
// based on subject, resource, action, and context information.
package authzen

// Entity represents a component in an AuthZEN evaluation request.
// It's used to represent subjects (who is performing the action),
// resources (what is being accessed), and can include properties like X.509 certificates.
type Entity struct {
	Type       string                 `json:"type"`                 // The entity type identifier
	ID         string                 `json:"id"`                   // The unique identifier for this entity
	Properties map[string]interface{} `json:"properties,omitempty"` // Additional properties, may include X.509 certificates as "x5c"
}

// EvaluationRequest represents an authorization decision request in the AuthZEN protocol.
// It follows the structure defined in the AuthZEN evaluation-request.schema.json schema.
// This request carries all information needed to make an authorization decision,
// including X.509 certificates that may be present in the properties or context.
type EvaluationRequest struct {
	Subject  Entity `json:"subject"`  // The entity attempting to perform an action
	Resource Entity `json:"resource"` // The entity being acted upon
	Action   struct {
		Name       string                 `json:"name"`                 // The name of the action being performed
		Properties map[string]interface{} `json:"properties,omitempty"` // Additional properties for the action
	} `json:"action"`
	Context map[string]interface{} `json:"context,omitempty"` // Additional contextual information
}

// EvaluationResponse represents the authorization decision response in the AuthZEN protocol.
// It follows the structure defined in the AuthZEN evaluation-response.schema.json schema.
// The response contains a boolean decision and optional context with reasons for the decision.
type EvaluationResponse struct {
	Decision bool `json:"decision"` // Whether the action is permitted (true) or denied (false)
	Context  *struct {
		ID          string                 `json:"id"`                     // An optional identifier for the decision
		ReasonAdmin map[string]interface{} `json:"reason_admin,omitempty"` // Detailed reason for administrators
		ReasonUser  map[string]interface{} `json:"reason_user,omitempty"`  // User-friendly reason message
	} `json:"context,omitempty"` // Optional context with decision details
}
