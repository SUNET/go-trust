package dsig

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	xmldsig "github.com/russellhaering/goxmldsig"
)

// FileSigner implements XMLSigner using certificate and private key files
type FileSigner struct {
	CertFile string
	KeyFile  string
}

// NewFileSigner creates a new FileSigner from certificate and key file paths
func NewFileSigner(certFile, keyFile string) *FileSigner {
	return &FileSigner{
		CertFile: certFile,
		KeyFile:  keyFile,
	}
}

// Sign implements XMLSigner.Sign using certificate and key files
func (fs *FileSigner) Sign(xmlData []byte) ([]byte, error) {
	// Load the certificate and private key
	certData, err := os.ReadFile(fs.CertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	keyData, err := os.ReadFile(fs.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Parse the certificate
	certBlock, _ := pem.Decode(certData)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Parse the private key
	keyBlock, _ := pem.Decode(keyData)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode key PEM")
	}

	// Try to parse the key in different formats
	var privateKey *rsa.PrivateKey
	var privateKeyAny interface{}

	// Try PKCS1 format
	privateKey, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		// Try PKCS8 format
		privateKeyAny, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		var ok bool
		privateKey, ok = privateKeyAny.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
	}

	// Create a key store from the loaded certificate and private key
	keyStore := &fileKeyStore{
		cert: cert,
		key:  privateKey,
	}

	return SignXMLWithKeyStore(xmlData, keyStore)
}

// fileKeyStore implements the xmldsig.X509KeyStore interface
type fileKeyStore struct {
	cert *x509.Certificate
	key  *rsa.PrivateKey
}

// GetKeyPair returns the private key and certificate for signing
func (ks *fileKeyStore) GetKeyPair() (*rsa.PrivateKey, []byte, error) {
	return ks.key, ks.cert.Raw, nil
}

// ToXMLDSigSigner converts a FileSigner to an xmldsig.Signer implementation
func (fs *FileSigner) ToXMLDSigSigner() (xmldsig.Signer, error) {
	// Load the certificate and private key
	certData, err := os.ReadFile(fs.CertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	keyData, err := os.ReadFile(fs.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Parse the certificate
	certBlock, _ := pem.Decode(certData)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Parse the private key
	keyBlock, _ := pem.Decode(keyData)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode key PEM")
	}

	// Try to parse the key in different formats
	var privateKey *rsa.PrivateKey
	var privateKeyAny interface{}

	// Try PKCS1 format
	privateKey, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		// Try PKCS8 format
		privateKeyAny, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		var ok bool
		privateKey, ok = privateKeyAny.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
	}

	// Use the file private key with certificate to create a new xmldsig.Signer
	// Default to SHA256 for the signing algorithm
	return xmldsig.NewFileSigner(privateKey, cert.Raw, crypto.SHA256)
}
