// Package authzen provides types and functions for the AuthZEN protocol.
// AuthZEN is an authorization protocol that allows for policy decisions
// based on subject, resource, action, and context information.
//
// This implementation follows the AuthZEN Trust Registry Profile as specified in
// draft-johansson-authzen-trust: https://leifj.github.io/draft-johansson-authzen-trust/
package authzen

import "fmt"

// Subject represents the name part of the name-to-key binding in a trust evaluation request.
// According to the AuthZEN Trust Registry Profile:
// - type MUST be the constant string "key"
// - id MUST be the name bound to the public key to be validated
// @Description Subject in an AuthZEN trust evaluation request
type Subject struct {
	Type string `json:"type" example:"key"`           // MUST be "key"
	ID   string `json:"id" example:"did:example:123"` // The name bound to the public key
}

// Resource represents the public key part of the name-to-key binding in a trust evaluation request.
// According to the AuthZEN Trust Registry Profile:
// - type MUST be one of "jwk" or "x5c"
// - id MUST be the same as subject.id
// - key MUST contain the public key in the format specified by type
// @Description Resource (public key) in an AuthZEN trust evaluation request
type Resource struct {
	Type string        `json:"type" example:"x5c"`             // MUST be "jwk" or "x5c"
	ID   string        `json:"id" example:"did:example:123"`   // MUST match subject.id
	Key  []interface{} `json:"key" swaggertype:"array,string"` // Public key data (JWK object or x5c array)
}

// Action represents the role associated with the name-to-key binding.
// This is optional and used to distinguish different uses of the same name-to-key binding.
// For example, to authorize that an X.509 certificate is allowed to act as a TLS server
// or as a digital credential issuer.
// @Description Action (role) in an AuthZEN trust evaluation request
type Action struct {
	Name string `json:"name" example:"http://ec.europa.eu/NS/wallet-provider"` // The role name
}

// EvaluationRequest represents a trust evaluation request according to the AuthZEN Trust Registry Profile.
// The client (PEP) requests that the server (PDP) authorizes the binding between the name
// specified by Subject and the public key specified by Resource. Optionally, Action constrains
// the authorization to a specific role.
// @Description AuthZEN trust evaluation request (draft-johansson-authzen-trust)
type EvaluationRequest struct {
	Subject  Subject                `json:"subject"`                                       // The name to be bound to the key
	Resource Resource               `json:"resource"`                                      // The public key to be validated
	Action   *Action                `json:"action,omitempty"`                              // Optional role constraint
	Context  map[string]interface{} `json:"context,omitempty" swaggertype:"object,string"` // Optional context (MUST NOT be critical)
}

// EvaluationResponse represents the authorization decision response in the AuthZEN protocol.
// This profile does not constrain or profile the standard AuthZEN response message format.
// @Description AuthZEN evaluation response with trust decision
type EvaluationResponse struct {
	Decision bool                       `json:"decision" example:"true"` // Whether the name-to-key binding is authorized
	Context  *EvaluationResponseContext `json:"context,omitempty"`       // Optional context with decision details
}

// EvaluationResponseContext contains additional information about an authorization decision
// @Description Context information for evaluation response
type EvaluationResponseContext struct {
	ID     string                 `json:"id,omitempty" example:"decision-123"`   // Optional identifier for the decision
	Reason map[string]interface{} `json:"reason,omitempty" swaggertype:"object"` // Reason information (user or admin)
}

// Validate checks if the EvaluationRequest is compliant with the AuthZEN Trust Registry Profile.
// Returns an error if the request doesn't meet the specification requirements.
func (r *EvaluationRequest) Validate() error {
	// Subject.type MUST be "key"
	if r.Subject.Type != "key" {
		return fmt.Errorf("subject.type must be 'key', got '%s'", r.Subject.Type)
	}

	// Subject.id MUST be present
	if r.Subject.ID == "" {
		return fmt.Errorf("subject.id must be present")
	}

	// Resource.type MUST be "jwk" or "x5c"
	if r.Resource.Type != "jwk" && r.Resource.Type != "x5c" {
		return fmt.Errorf("resource.type must be 'jwk' or 'x5c', got '%s'", r.Resource.Type)
	}

	// Resource.id MUST be present and MUST match subject.id
	if r.Resource.ID == "" {
		return fmt.Errorf("resource.id must be present")
	}
	if r.Resource.ID != r.Subject.ID {
		return fmt.Errorf("resource.id (%s) must match subject.id (%s)", r.Resource.ID, r.Subject.ID)
	}

	// Resource.key MUST be present
	if len(r.Resource.Key) == 0 {
		return fmt.Errorf("resource.key must be present and non-empty")
	}

	return nil
}
