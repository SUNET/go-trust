// Package x509util provides X.509 certificate parsing utilities.
// This package contains shared certificate parsing functions used across
// go-trust components, particularly for extracting certificates from
// AuthZEN resource.key arrays in various formats (x5c arrays, JWK objects).
package x509util

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// ParseX5CFromArray parses X.509 certificates from an array of base64-encoded strings.
//
// This function expects the input format used by AuthZEN resource.key when
// resource.type is "x5c": an array of base64-encoded DER certificates.
//
// Parameters:
//   - key: Array of interface{} values that should be base64-encoded certificate strings
//
// Returns:
//   - Slice of parsed X.509 certificates (leaf certificate first)
//   - Error if any certificate fails to decode or parse
//
// Example input:
//
//	[]interface{}{
//	    "MIIC...base64...==",  // leaf certificate
//	    "MIID...base64...==",  // intermediate CA
//	}
func ParseX5CFromArray(key []interface{}) ([]*x509.Certificate, error) {
	certs := make([]*x509.Certificate, 0, len(key))

	for i, item := range key {
		certStr, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("resource.key[%d] is not a string", i)
		}

		// Decode base64
		certDER, err := base64.StdEncoding.DecodeString(certStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode resource.key[%d]: %w", i, err)
		}

		// Parse certificate
		cert, err := x509.ParseCertificate(certDER)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate at resource.key[%d]: %w", i, err)
		}

		certs = append(certs, cert)
	}

	return certs, nil
}

// ParseX5CFromJWK extracts X.509 certificates from a JWK object's x5c claim.
//
// This function expects the input format used by AuthZEN resource.key when
// resource.type is "jwk": an array containing a single JWK object with an
// x5c (X.509 Certificate Chain) claim.
//
// Parameters:
//   - key: Array containing a single JWK object (map[string]interface{})
//
// Returns:
//   - Slice of parsed X.509 certificates from the x5c claim
//   - Error if JWK format is invalid or x5c claim is missing/malformed
//
// Example input:
//
//	[]interface{}{
//	    map[string]interface{}{
//	        "kty": "RSA",
//	        "n": "...",
//	        "e": "AQAB",
//	        "x5c": []interface{}{"MIIC...==", "MIID...=="},
//	    },
//	}
func ParseX5CFromJWK(key []interface{}) ([]*x509.Certificate, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("resource.key is empty")
	}

	// resource.key should contain a single JWK object
	jwkMap, ok := key[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("resource.key[0] is not a JWK object")
	}

	// Extract x5c claim
	x5cRaw, ok := jwkMap["x5c"]
	if !ok {
		return nil, fmt.Errorf("JWK does not contain x5c claim")
	}

	x5cArray, ok := x5cRaw.([]interface{})
	if !ok {
		// Try to handle it as a JSON array string (some implementations serialize it)
		if x5cStr, ok := x5cRaw.(string); ok {
			var arr []interface{}
			if err := json.Unmarshal([]byte(x5cStr), &arr); err == nil {
				x5cArray = arr
			} else {
				return nil, fmt.Errorf("JWK x5c is not an array or valid JSON: %w", err)
			}
		} else {
			return nil, fmt.Errorf("JWK x5c is not an array")
		}
	}

	// Parse certificates from x5c array
	return ParseX5CFromArray(x5cArray)
}
