package dsig

import (
	"crypto/rsa"

	"github.com/beevik/etree"
	xmldsig "github.com/russellhaering/goxmldsig"
)

// XMLSigner represents an interface for signing XML documents with XML-DSIG
type XMLSigner interface {
	// Sign takes XML data and returns signed XML data
	Sign(xmlData []byte) ([]byte, error)
}

// X509KeyStore is a wrapper around the goxmldsig X509KeyStore interface
type X509KeyStore interface {
	GetKeyPair() (*rsa.PrivateKey, []byte, error)
}

// SignXML signs XML data using any implementation of the xmldsig.Signer interface
func SignXML(xmlData []byte, signer xmldsig.Signer) ([]byte, error) {
	// Create the signing context with our signer
	ctx := xmldsig.NewDefaultSigningContextWithSigner(signer)

	// Use exclusive canonicalization (C14N)
	ctx.Canonicalizer = xmldsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")

	// Parse the XML document
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlData); err != nil {
		return nil, err
	}

	// Sign the XML document
	signedDoc, err := ctx.SignEnveloped(doc.Root())
	if err != nil {
		return nil, err
	}

	// Return the signed XML
	doc2 := etree.NewDocument()
	doc2.SetRoot(signedDoc)
	return doc2.WriteToBytes()
}

// SignXMLWithKeyStore signs XML data using the provided X509KeyStore
func SignXMLWithKeyStore(xmlData []byte, keyStore xmldsig.X509KeyStore) ([]byte, error) {
	// Create the signature context
	ctx := xmldsig.NewDefaultSigningContext(keyStore)
	ctx.Canonicalizer = xmldsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")

	// Parse the XML document
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlData); err != nil {
		return nil, err
	}

	// Sign the XML document
	signedDoc, err := ctx.SignEnveloped(doc.Root())
	if err != nil {
		return nil, err
	}

	// Return the signed XML
	doc2 := etree.NewDocument()
	doc2.SetRoot(signedDoc)
	return doc2.WriteToBytes()
}

// GetSigningMethodName returns a string description of the default signing method
func GetSigningMethodName() string {
	return "rsa-sha256" // Default to SHA256
}
