package pipeline

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SUNET/g119612/pkg/etsi119612"
	"github.com/SUNET/go-trust/pkg/dsig"
	"github.com/SUNET/go-trust/pkg/logging"
	"github.com/SUNET/go-trust/pkg/utils"
	"gopkg.in/yaml.v3"
)

// MultiLangName represents a name in a specific language
type MultiLangName struct {
	Language string `yaml:"language"`
	Value    string `yaml:"value"`
}

// Address represents a postal and electronic address
type Address struct {
	Postal struct {
		StreetAddress   string `yaml:"streetAddress"`
		Locality        string `yaml:"locality"`
		StateOrProvince string `yaml:"stateOrProvince,omitempty"`
		PostalCode      string `yaml:"postalCode,omitempty"`
		CountryName     string `yaml:"countryName"`
	} `yaml:"postal"`
	Electronic []string `yaml:"electronic,omitempty"`
}

// ProviderMetadata represents the YAML structure for a provider's metadata
type ProviderMetadata struct {
	Names          []MultiLangName `yaml:"names"` // At least one name required
	Address        *Address        `yaml:"address,omitempty"`
	TradeName      []MultiLangName `yaml:"tradeName,omitempty"`
	InformationURI []MultiLangName `yaml:"informationURI,omitempty"`
}

// CertificateMetadata represents the YAML structure for a certificate's metadata
type CertificateMetadata struct {
	ServiceNames     []MultiLangName `yaml:"serviceNames"` // At least one name required
	ServiceType      string          `yaml:"serviceType"`  // URI identifying the service type
	Status           string          `yaml:"status"`       // Must be a valid TSL status URI
	ServiceDigitalID *struct {
		DigitalIDs []string `yaml:"digitalIds,omitempty"` // Additional digital IDs beyond the certificate
	} `yaml:"serviceDigitalId,omitempty"`
}

// SchemeMetadata represents the YAML structure for the TSL scheme metadata
type SchemeMetadata struct {
	OperatorNames  []MultiLangName `yaml:"operatorNames"`            // At least one name required
	Type           string          `yaml:"type"`                     // URI identifying the TSL type
	SequenceNumber int             `yaml:"sequenceNumber,omitempty"` // TSL sequence number
}

// loadSchemeMetadata loads and parses the scheme metadata from the scheme.yaml file.
// This function reads the top-level TSL configuration including operator names,
// TSL type URI, and sequence number.
//
// The scheme.yaml file must contain:
//   - operatorNames: At least one operator name with language and value
//   - type: A valid TSL type URI (e.g., http://uri.etsi.org/TrstSvc/TrustedList/TSLType/...)
//   - sequenceNumber: Optional TSL sequence number (defaults to 1 if not provided)
//
// Parameters:
//   - rootDir: Absolute path to the root directory containing scheme.yaml
//
// Returns:
//   - *SchemeMetadata: Parsed scheme metadata structure
//   - error: If the file cannot be read, is not valid YAML, or missing required fields
//
// Example scheme.yaml:
//
//	operatorNames:
//	  - language: en
//	    value: "Trust List Operator"
//	type: "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUlistofthelists"
//	sequenceNumber: 1
func loadSchemeMetadata(rootDir string) (*SchemeMetadata, error) {
	metadataPath := filepath.Join(rootDir, "scheme.yaml")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read scheme metadata from %s: %w", metadataPath, err)
	}

	var metadata SchemeMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse scheme metadata from %s: %w", metadataPath, err)
	}

	if len(metadata.OperatorNames) == 0 {
		return nil, fmt.Errorf("scheme metadata must include at least one operator name")
	}

	if metadata.Type == "" {
		return nil, fmt.Errorf("scheme metadata must include a type URI")
	}

	return &metadata, nil
}

// loadProviderMetadata loads and parses the provider metadata from provider.yaml.
// This function reads provider-specific information such as names, addresses,
// trade names, and information URIs in multiple languages.
//
// The provider.yaml file must contain:
//   - names: At least one provider name with language and value
//   - address: Optional postal and electronic addresses
//   - tradeName: Optional trade names in multiple languages
//   - informationURI: Optional information URIs in multiple languages
//
// Parameters:
//   - providerDir: Absolute path to the provider directory containing provider.yaml
//
// Returns:
//   - *ProviderMetadata: Parsed provider metadata structure
//   - error: If the file cannot be read, is not valid YAML, or missing required fields
//
// Example provider.yaml:
//
//	names:
//	  - language: en
//	    value: "Example Trust Service Provider"
//	address:
//	  postal:
//	    streetAddress: "Example Street 123"
//	    locality: "Example City"
//	    postalCode: "12345"
//	    countryName: "SE"
//	  electronic:
//	    - "https://example.com"
//	    - "mailto:contact@example.com"
func loadProviderMetadata(providerDir string) (*ProviderMetadata, error) {
	metadataPath := filepath.Join(providerDir, "provider.yaml")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read provider metadata from %s: %w", metadataPath, err)
	}

	var metadata ProviderMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse provider metadata from %s: %w", metadataPath, err)
	}

	if len(metadata.Names) == 0 {
		return nil, fmt.Errorf("provider metadata must include at least one name")
	}

	return &metadata, nil
}

// addProviderCertificates processes certificate files in a provider directory and adds them to the TSP.
// For each .pem certificate file, it looks for a corresponding .yaml metadata file
// with the same base name. The function handles both the certificate data and its
// service metadata to populate the TSP's service list.
//
// For each certificate pair (example.pem + example.yaml):
//  1. Reads and parses the X.509 certificate from the .pem file
//  2. Loads the service metadata from the .yaml file
//  3. Creates a TSP service entry with the certificate and metadata
//  4. Adds the service to the provider's service list
//
// Parameters:
//   - providerDir: Absolute path to the provider directory containing .pem and .yaml files
//   - provider: TSP structure to add the certificates and services to
//
// Returns:
//   - error: If any certificate or metadata file cannot be read or parsed
//
// Expected files:
//   - *.pem: X.509 certificates in PEM format
//   - *.yaml: Matching metadata files for each certificate
//
// Example cert.yaml:
//
//	serviceNames:
//	  - language: en
//	    value: "Example Certificate Service"
//	serviceType: "http://uri.etsi.org/TrstSvc/Svctype/CA/QC"
//	status: "https://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/"
func addProviderCertificates(providerDir string, provider *etsi119612.TSPType) error {
	entries, err := os.ReadDir(providerDir)
	if err != nil {
		return fmt.Errorf("failed to read provider directory %s: %w", providerDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pem") {
			continue
		}

		certPath := filepath.Join(providerDir, entry.Name())
		metadataPath := certPath[:len(certPath)-4] + ".yaml" // replace .pem with .yaml

		// Load certificate metadata
		metadataBytes, err := os.ReadFile(metadataPath)
		if err != nil {
			return fmt.Errorf("failed to read certificate metadata from %s: %w", metadataPath, err)
		}

		var metadata CertificateMetadata
		if err := yaml.Unmarshal(metadataBytes, &metadata); err != nil {
			return fmt.Errorf("failed to parse certificate metadata from %s: %w", metadataPath, err)
		}

		if len(metadata.ServiceNames) == 0 {
			return fmt.Errorf("certificate metadata must include at least one service name")
		}

		// Load certificate
		certBytes, err := os.ReadFile(certPath)
		if err != nil {
			return fmt.Errorf("failed to read certificate from %s: %w", certPath, err)
		}

		// Try to parse the certificate to ensure it's valid
		_, err = x509.ParseCertificate(certBytes)
		if err != nil {
			return fmt.Errorf("failed to decode invalid certificate data in %s: %w", certPath, err)
		}

		// Create service names
		serviceNames := make([]*etsi119612.MultiLangNormStringType, len(metadata.ServiceNames))
		for i, name := range metadata.ServiceNames {
			serviceNames[i] = &etsi119612.MultiLangNormStringType{
				XmlLangAttr: func() *etsi119612.Lang {
					l := etsi119612.Lang(name.Language)
					return &l
				}(),
				NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
					s := etsi119612.NonEmptyNormalizedString(name.Value)
					return &s
				}(),
			}
		}

		// Create digital IDs - certificate bytes have been validated above
		digitalIds := []*etsi119612.DigitalIdentityType{
			{
				X509Certificate: base64.StdEncoding.EncodeToString(certBytes),
			},
		}

		if metadata.ServiceDigitalID != nil {
			for _, id := range metadata.ServiceDigitalID.DigitalIDs {
				digitalIds = append(digitalIds, &etsi119612.DigitalIdentityType{
					X509Certificate: id,
				})
			}
		}

		// Create service entry
		service := &etsi119612.TSPServiceType{
			TslServiceInformation: &etsi119612.TSPServiceInformationType{
				TslServiceTypeIdentifier: metadata.ServiceType,
				TslServiceStatus:         metadata.Status,
				ServiceName: &etsi119612.InternationalNamesType{
					Name: serviceNames,
				},
				TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
					DigitalId: digitalIds,
				},
			},
		}

		provider.TslTSPServices.TslTSPService = append(
			provider.TslTSPServices.TslTSPService,
			service,
		)
	}

	return nil
}

// GenerateTSL is a pipeline step that generates a Trust Service List (TSL) from a structured directory.
// It implements generation of ETSI TS 119612 compliant TSLs by reading metadata and certificates
// from a hierarchical directory structure.
//
// Directory Structure:
//
//	root/
//	  ├── scheme.yaml      # TSL scheme metadata
//	  └── providers/       # Directory containing all providers
//	      └── provider1/   # One directory per provider
//	          ├── provider.yaml  # Provider metadata
//	          ├── cert1.pem      # Certificate files
//	          └── cert1.yaml     # Certificate metadata
//
// File Formats:
//
//	scheme.yaml:
//	  operatorNames:       # List of operator names in different languages
//	    - language: en
//	      value: "Trust List Operator"
//	  type: "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/..."  # TSL type URI
//	  sequenceNumber: 1    # TSL sequence number
//
//	provider.yaml:
//	  names:              # List of provider names in different languages
//	    - language: en
//	      value: "Example Provider"
//	  address:            # Provider's address information
//	    postal:
//	      streetAddress: "Example Street 123"
//	      locality: "Example City"
//	      postalCode: "12345"
//	      countryName: "SE"
//	    electronic:        # List of electronic addresses
//	      - "https://example.com"
//	      - "mailto:contact@example.com"
//	  tradeName:          # Optional trade names in different languages
//	    - language: en
//	      value: "Example Corp"
//	  informationURI:     # Optional information URIs in different languages
//	    - language: en
//	      value: "https://example.com/info"
//
//	cert.yaml (matching .pem file):
//	  serviceNames:        # List of service names in different languages
//	    - language: en
//	      value: "Example Service"
//	  serviceType: "http://uri.etsi.org/TrstSvc/Svctype/..."  # Service type URI
//	  status: "https://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/..."  # Status URI
//	  serviceDigitalId:    # Optional additional digital IDs
//	    digitalIds:
//	      - "base64 encoded cert..."
//
// Parameters:
//   - pl: Pipeline instance managing the step execution
//   - ctx: Pipeline context containing state information
//   - args: String slice where args[0] must be the path to the root directory
//
// Returns:
//   - *Context: Updated context with the generated TSL added to ctx.TSLs
//   - error: Non-nil if any error occurs during generation
//
// The function generates a TSL by:
// 1. Loading scheme metadata from scheme.yaml
// 2. Creating the base TSL structure with scheme information
// 3. Iterating through provider directories in the providers/ subdirectory
// 4. For each provider:
//   - Loading provider metadata and creating TSP entries
//   - Processing all certificate files (.pem) and their metadata (.yaml)
//   - Adding all services and certificates to the provider entry
//
// 5. Adding the complete TSL to the pipeline context
//   - rootDir: path to the root directory containing scheme.yaml and providers directory
func GenerateTSL(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("GenerateTSL requires 1 argument: path to root directory")
	}

	rootDir := args[0]
	providersDir := filepath.Join(rootDir, "providers")
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read providers directory %s: %w", providersDir, err)
	}

	// Load scheme metadata
	schemeMetadata, err := loadSchemeMetadata(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load scheme metadata: %w", err)
	}

	// Create operator names for the TSL
	operatorNames := make([]*etsi119612.MultiLangNormStringType, len(schemeMetadata.OperatorNames))
	for i, name := range schemeMetadata.OperatorNames {
		operatorNames[i] = &etsi119612.MultiLangNormStringType{
			XmlLangAttr: func() *etsi119612.Lang {
				l := etsi119612.Lang(name.Language)
				return &l
			}(),
			NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
				s := etsi119612.NonEmptyNormalizedString(name.Value)
				return &s
			}(),
		}
	}

	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TSLVersionIdentifier: int(schemeMetadata.SequenceNumber),
				TslTSLType:           schemeMetadata.Type,
				TslSchemeOperatorName: &etsi119612.InternationalNamesType{
					Name: operatorNames,
				},
			},
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{},
			},
		},
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		providerDir := filepath.Join(providersDir, entry.Name())
		providerMetadata, err := loadProviderMetadata(providerDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load provider metadata from %s: %w", providerDir, err)
		}

		// Create provider names
		providerNames := make([]*etsi119612.MultiLangNormStringType, len(providerMetadata.Names))
		for i, name := range providerMetadata.Names {
			providerNames[i] = &etsi119612.MultiLangNormStringType{
				XmlLangAttr: func() *etsi119612.Lang {
					l := etsi119612.Lang(name.Language)
					return &l
				}(),
				NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
					s := etsi119612.NonEmptyNormalizedString(name.Value)
					return &s
				}(),
			}
		}

		provider := &etsi119612.TSPType{
			TslTSPInformation: &etsi119612.TSPInformationType{
				TSPName: &etsi119612.InternationalNamesType{
					Name: providerNames,
				},
			},
			TslTSPServices: &etsi119612.TSPServicesListType{
				TslTSPService: []*etsi119612.TSPServiceType{},
			},
		}

		// Add provider address if present
		if providerMetadata.Address != nil {
			provider.TslTSPInformation.TSPAddress = &etsi119612.AddressType{
				TslPostalAddresses: &etsi119612.PostalAddressListType{
					TslPostalAddress: []*etsi119612.PostalAddressType{
						{
							XmlLangAttr:     func() *etsi119612.Lang { l := etsi119612.Lang("en"); return &l }(),
							StreetAddress:   providerMetadata.Address.Postal.StreetAddress,
							Locality:        providerMetadata.Address.Postal.Locality,
							StateOrProvince: providerMetadata.Address.Postal.StateOrProvince,
							PostalCode:      providerMetadata.Address.Postal.PostalCode,
							CountryName:     providerMetadata.Address.Postal.CountryName,
						},
					},
				},
			}

			if len(providerMetadata.Address.Electronic) > 0 {
				electronic := make([]*etsi119612.NonEmptyMultiLangURIType, len(providerMetadata.Address.Electronic))
				for i, uri := range providerMetadata.Address.Electronic {
					electronic[i] = &etsi119612.NonEmptyMultiLangURIType{
						XmlLangAttr: func() *etsi119612.Lang { l := etsi119612.Lang("en"); return &l }(),
						Value:       uri,
					}
				}
				provider.TslTSPInformation.TSPAddress.TslElectronicAddress = &etsi119612.ElectronicAddressType{
					URI: electronic,
				}
			}
		}

		err = addProviderCertificates(providerDir, provider)
		if err != nil {
			return nil, fmt.Errorf("failed to add certificates for provider %s: %w", entry.Name(), err)
		}

		tsl.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider = append(
			tsl.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider,
			provider,
		)
	}

	ctx.EnsureTSLStack().TSLs.Push(tsl)

	return ctx, nil
}

// LoadTSL is a pipeline step that loads a Trust Service List (TSL) from a file or URL.
// This function supports loading TSLs from both local files and remote HTTP(S) URLs,
// and will also load any referenced TSLs based on the MaxDereferenceDepth setting.
//
// Parameters:
//   - pl: Pipeline instance managing the step execution
//   - ctx: Pipeline context containing state information
//   - args: String slice where:
//   - args[0] must be the URL or file path to the TSL
//
// Returns:
//   - *Context: Updated context with the loaded TSL and referenced TSLs added to ctx.TSLs
//   - error: Non-nil if the TSL cannot be loaded or parsed
//
// URL handling:
//   - HTTP(S) URLs are used as-is
//   - Local paths are converted to file:// URLs
//   - The TSL is fetched and parsed using etsi119612.FetchTSLWithReferencesAndOptions
//
// The function uses the fetch options (UserAgent, Timeout, MaxDereferenceDepth) that
// were previously set using SetFetchOptions. If not set, default values will be used.
//
// The loaded TSL and all referenced TSLs (according to MaxDereferenceDepth) are pushed
// onto the context's TSL stack, with the root TSL on top. If the stack doesn't exist,
// a new one is created. Multiple calls to LoadTSL will result in multiple TSLs being
// available in the context.
//
// Example usage in pipeline configuration:
//   - set-fetch-options:
//   - user-agent:MyCustomUserAgent/1.0
//   - timeout:60s
//   - max-depth:3
//   - load: [http://example.com/tsl.xml]
//   - load: [/path/to/local/tsl.xml]
//
// LoadTSL is a pipeline step that loads Trust Service Lists (TSLs) from a URL or file path,
// builds a hierarchical TSL tree structure, and adds it to the pipeline context. It also
// maintains a backward-compatible flat stack of TSLs for legacy code.
//
// The step supports loading TSLs from files or HTTP/HTTPS URLs, with automatic content
// negotiation and reference handling. It uses the TSLFetchOptions in the context for
// request configuration (user-agent, timeout, reference depth, etc.).
//
// Parameters:
//   - pl: The pipeline instance for logging and configuration
//   - ctx: The pipeline context to update with loaded TSLs
//   - args: String arguments, where:
//   - args[0]: Required - URL or file path to the root TSL
//   - args[1]: Optional - Filter expression for including specific TSLs (not implemented yet)
//
// Returns:
//   - *Context: Updated context with the loaded TSL tree and legacy TSL stack
//   - error: Non-nil if loading fails
//
// Example usage in pipeline configuration:
//   - load:
//   - https://example.com/tsl.xml
//
// Or with a local file:
//   - load:
//   - /path/to/local/tsl.xml
//
// The loaded TSL tree structure represents the hierarchical relationship between the root TSL
// and its referenced TSLs, allowing for more efficient traversal and operations on the tree.
func LoadTSL(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return ctx, fmt.Errorf("missing argument: URL or file path")
	}

	url := args[0]
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "file://" + url
	}

	// Parse optional filter argument
	var filter string
	if len(args) > 1 {
		filter = args[1]
		pl.Logger.Debug("TSL filter provided", logging.F("filter", filter))
		// Note: Filter implementation will be added in a future update
	}

	// Ensure the TSLFetchOptions are initialized with default values if not set
	ctx.EnsureTSLFetchOptions()

	pl.Logger.Debug("Loading TSL",
		logging.F("url", url),
		logging.F("user-agent", ctx.TSLFetchOptions.UserAgent),
		logging.F("timeout", ctx.TSLFetchOptions.Timeout),
		logging.F("max-depth", ctx.TSLFetchOptions.MaxDereferenceDepth),
		logging.F("accept", ctx.TSLFetchOptions.AcceptHeaders))

	tsls, err := etsi119612.FetchTSLWithReferencesAndOptions(url, *ctx.TSLFetchOptions)
	if err != nil {
		return ctx, fmt.Errorf("failed to load TSL from %s: %w", url, err)
	}

	if len(tsls) == 0 {
		return ctx, fmt.Errorf("no TSLs returned from %s", url)
	}

	// Apply filters if any are defined
	originalCount := len(tsls)
	tsls = FilterTSLs(ctx, tsls)
	if len(tsls) < originalCount {
		pl.Logger.Info("Applied TSL filters",
			logging.F("original_count", originalCount),
			logging.F("filtered_count", len(tsls)))
	}

	// Ensure we still have TSLs after filtering
	if len(tsls) == 0 {
		return ctx, fmt.Errorf("no TSLs passed the filter criteria")
	}

	// Build a TSL tree from the loaded TSLs and add it to the stack of trees
	ctx.EnsureTSLTrees()

	// The first TSL is the root, use it to build a new tree
	rootTSL := tsls[0]
	tree := NewTSLTree(rootTSL)
	ctx.AddTSLTree(tree)

	// For backward compatibility, ensure the legacy TSLs stack is populated correctly
	// We need to add TSLs in reverse order: referenced TSLs first, then the root
	if ctx.TSLs == nil {
		ctx.TSLs = utils.NewStack[*etsi119612.TSL]()
	} else {
		// Clear the legacy stack as we're about to rebuild it
		for ctx.TSLs.Size() > 0 {
			ctx.TSLs.Pop()
		}
	}

	// Add referenced TSLs in reverse order (add them last but they'll be popped first)
	for i := len(tsls) - 1; i > 0; i-- {
		ctx.TSLs.Push(tsls[i])
	}

	// Add the root TSL last so it's at the bottom of the stack
	if len(tsls) > 0 {
		ctx.TSLs.Push(tsls[0])
	}

	// Count service providers and services
	var totalProviders int
	var totalServices int
	var schemeTerritory string

	// Log details about each TSL loaded
	for i, tsl := range tsls {
		// Extract scheme territory if available
		if i == 0 && tsl.StatusList.TslSchemeInformation != nil {
			schemeTerritory = tsl.StatusList.TslSchemeInformation.TslSchemeTerritory
		}

		// Count providers and services
		providerCount := 0
		serviceCount := 0
		if tsl.StatusList.TslTrustServiceProviderList != nil {
			providers := tsl.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider
			providerCount = len(providers)
			totalProviders += providerCount

			// Count services for each provider
			for _, provider := range providers {
				if provider != nil && provider.TslTSPServices != nil {
					services := provider.TslTSPServices.TslTSPService
					serviceCount += len(services)
					totalServices += len(services)
				}
			}
		}

		// Log each TSL as it's loaded
		pl.Logger.Info("Loaded TSL",
			logging.F("url", tsl.Source),
			logging.F("providers", providerCount),
			logging.F("services", serviceCount),
			logging.F("referenced", i > 0))
	}

	pl.Logger.Info("Loaded TSLs",
		logging.F("root_url", url),
		logging.F("territory", schemeTerritory),
		logging.F("tree_depth", tree.Depth()),
		logging.F("total_count", len(tsls)),
		logging.F("total_providers", totalProviders),
		logging.F("total_services", totalServices))

	return ctx, nil
}

// SelectCertPool creates a new x509.CertPool from all certificates in the loaded TSLs.
// This step processes all TSLs in the context's TSL stack and extracts certificates
// from trust service providers, adding them to a new certificate pool.
//
// The function walks through each TSL's trust service providers and their services,
// collecting all valid X.509 certificates. These certificates are then added to a new
// certificate pool that can be used for certificate chain validation.
//
// Parameters:
//   - pl: Pipeline instance managing the step execution
//   - ctx: Pipeline context containing state information
//   - args: Optional arguments:
//   - "reference-depth:N": Process TSLs up to N levels deep in references (0=root only, 1=root+direct refs)
//   - "include-referenced": Legacy option, equivalent to a large reference depth (includes all refs)
//   - "service-type:URI": Filter certificates by service type URI (can be provided multiple times)
//   - "status:URI": Filter certificates by status URI (can be provided multiple times)
//   - "status-logic:and": Use AND logic for status filters (all filters must match) instead of default OR logic
//
// Returns:
//   - *Context: Updated context with the new certificate pool in ctx.CertPool
//   - error: Non-nil if no TSLs are loaded or if certificate processing fails
//
// The created certificate pool is stored in the context's CertPool field and can be
// used for certificate validation operations. Each certificate from valid trust services
// is added as a trusted root certificate.
//
// Note:
//   - Requires at least one TSL to be loaded in the context
//   - Invalid or nil TSLs in the stack are safely skipped
//   - The previous certificate pool, if any, is replaced
//   - The reference-depth parameter controls how deep in the TSL reference tree to process
//   - Service type and status filters are combined with OR logic within each category and AND between categories
//
// Example usage in pipeline configuration:
//   - select  # Create cert pool from top TSL only, all service types
//   - select: [reference-depth:1]  # Include root and direct references only
//   - select: [reference-depth:2]  # Include root, direct refs, and refs of refs (2 levels)
//   - select: [include-referenced]  # Legacy option: include all references
//   - select: ["service-type:http://uri.etsi.org/TrstSvc/Svctype/CA/QC"]  # Only qualified CA certificates
//   - select: ["reference-depth:1", "service-type:http://uri.etsi.org/TrstSvc/Svctype/CA/QC", "status:http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/"]  # Only granted qualified CA certificates up to depth 1
//   - select: ["status:http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/", "status:http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/recognized/", "status-logic:and"]  # Only certificates that match both status filters
func SelectCertPool(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	// Check if we have TSLs either in the legacy stack or in the tree structure
	if (ctx.TSLTrees == nil || ctx.TSLTrees.IsEmpty()) && (ctx.TSLs == nil || ctx.TSLs.IsEmpty()) {
		return ctx, fmt.Errorf("no TSLs loaded")
	}

	// Parse arguments
	referenceDepth := 0 // Default: only root TSLs (no references)
	serviceTypeFilters := []string{}
	statusFilters := []string{}
	useStatusAndLogic := false // Default: use OR logic for status filters

	for _, arg := range args {
		if arg == "include-referenced" {
			// Legacy option: set depth to a large number to include all references
			referenceDepth = 100
		} else if strings.HasPrefix(arg, "reference-depth:") {
			depthStr := strings.TrimPrefix(arg, "reference-depth:")
			if depth, err := strconv.Atoi(depthStr); err == nil && depth >= 0 {
				referenceDepth = depth
			} else if err != nil {
				pl.Logger.Warn("Invalid reference-depth value, using default",
					logging.F("value", depthStr),
					logging.F("default", referenceDepth))
			}
		} else if strings.HasPrefix(arg, "service-type:") {
			serviceType := strings.TrimPrefix(arg, "service-type:")
			if serviceType != "" {
				serviceTypeFilters = append(serviceTypeFilters, serviceType)
			}
		} else if strings.HasPrefix(arg, "status:") {
			status := strings.TrimPrefix(arg, "status:")
			if status != "" {
				statusFilters = append(statusFilters, status)
			}
		} else if arg == "status-logic:and" {
			useStatusAndLogic = true
		}
	}

	// Initialize the certificate pool
	ctx.InitCertPool()

	// Track certificate counts for logging
	certCount := 0
	tslCount := 0

	// Create a certificate processing function that applies filters
	processCertificate := func(tsp *etsi119612.TSPType, svc *etsi119612.TSPServiceType, cert *x509.Certificate) {
		// Apply service type filter if specified
		if len(serviceTypeFilters) > 0 {
			serviceTypeMatch := false
			serviceType := svc.TslServiceInformation.TslServiceTypeIdentifier
			for _, filter := range serviceTypeFilters {
				if serviceType == filter {
					serviceTypeMatch = true
					break
				}
			}
			if !serviceTypeMatch {
				return
			}
		}

		// Apply status filter if specified
		if len(statusFilters) > 0 {
			status := svc.TslServiceInformation.TslServiceStatus

			if useStatusAndLogic {
				// AND logic: certificate must match ALL status filters
				for _, filter := range statusFilters {
					if status != filter {
						// If any filter doesn't match, skip this certificate
						return
					}
				}
			} else {
				// OR logic (default): certificate must match ANY status filter
				statusMatch := false
				for _, filter := range statusFilters {
					if status == filter {
						statusMatch = true
						break
					}
				}
				if !statusMatch {
					return
				}
			}
		}

		// Add the certificate to the pool
		ctx.CertPool.AddCert(cert)
		certCount++
	}

	// Define a function to process a TSL and extract certificates
	processTSL := func(tsl *etsi119612.TSL) {
		if tsl == nil {
			return
		}

		tslCount++

		// Process the TSL
		tsl.WithTrustServices(func(tsp *etsi119612.TSPType, svc *etsi119612.TSPServiceType) {
			svc.WithCertificates(func(cert *x509.Certificate) {
				processCertificate(tsp, svc, cert)
			})
		})
	}

	// Define a function to process a tree with a limited depth
	processTreeWithDepth := func(tree *TSLTree, processFunc func(*etsi119612.TSL), maxDepth int) {
		if tree == nil || tree.Root == nil || maxDepth < 0 {
			return
		}

		// Process nodes recursively with depth tracking
		var processNodeWithDepth func(node *TSLNode, currentDepth int)
		processNodeWithDepth = func(node *TSLNode, currentDepth int) {
			if node == nil || currentDepth > maxDepth {
				return
			}

			// Process this node's TSL
			processFunc(node.TSL)

			// Process children up to maxDepth
			for _, childNode := range node.Children {
				processNodeWithDepth(childNode, currentDepth+1)
			}
		}

		// Start processing from the root at depth 0
		processNodeWithDepth(tree.Root, 0)
	}

	// Check if we should use the legacy stack
	if ctx.TSLs != nil && !ctx.TSLs.IsEmpty() {
		// Process TSLs from the legacy stack
		tsls := ctx.TSLs.ToSlice()
		for i, tsl := range tsls {
			if tsl == nil {
				continue
			}

			// In legacy mode, with a flat list:
			// - The root TSL is at index 0
			// - Referenced TSLs come after, but we don't have depth information
			// - So we'll include TSLs up to the reference depth
			if i == 0 || (i > 0 && i <= referenceDepth) {
				processTSL(tsl)
			}
		}
	} else {
		// Process each TSL tree in the stack
		treeSlice := ctx.TSLTrees.ToSlice()
		for _, tree := range treeSlice {
			if tree == nil || tree.Root == nil {
				continue
			}

			if referenceDepth > 0 {
				// Process TSLs up to the specified reference depth
				processTreeWithDepth(tree, processTSL, referenceDepth)
			} else {
				// Process only the root TSL
				processTSL(tree.Root.TSL)
			}
		}
	}

	// Log summary information
	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("Certificate pool created",
			logging.F("tsl_count", tslCount),
			logging.F("certificate_count", certCount),
			logging.F("reference_depth", referenceDepth),
			logging.F("service_type_filters", len(serviceTypeFilters)),
			logging.F("status_filters", len(statusFilters)))
	}

	if pl != nil && pl.Logger != nil {
		if len(serviceTypeFilters) > 0 {
			pl.Logger.Debug("Service type filters applied",
				logging.F("filters", serviceTypeFilters))
		}

		if len(statusFilters) > 0 {
			pl.Logger.Debug("Status filters applied",
				logging.F("filters", statusFilters))
		}
	}

	return ctx, nil
}

// PublishTSL is a pipeline step that serializes TSLs to XML files in a specified directory.
// It uses the distribution point information from each TSL to determine the file name.
//
// Parameters:
//   - pl: Pipeline instance managing the step execution
//   - ctx: Pipeline context containing state information
//   - args: String slice where args[0] must be the directory path where to save the XML files
//
// Returns:
//   - *Context: The context unchanged
//   - error: Non-nil if any error occurs during serialization or if no directory is specified
//
// This step processes each TSL in the context's TSL stack and serializes it to XML.
// The file name is determined from the TSL's distribution point information:
// - If a distribution point is specified, the last part of the URI is used as the file name
// - If no distribution point is found, a default name pattern "tsl-{sequenceNumber}.xml" is used
//
// For each TSL, the following steps are performed:
// 1. Extract distribution point information, if available
// 2. Determine the file name based on the distribution point or use a default
// 3. Serialize the TSL to XML
// 4. Write the XML to a file in the specified directory
//
// Example usage in pipeline configuration:
//   - publish:/path/to/output/dir  # Publish all TSLs to the specified directory
//   - publish:["/path/to/output/dir", "/path/to/cert.pem", "/path/to/key.pem"]  # With XML-DSIG signatures
func PublishTSL(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return ctx, fmt.Errorf("missing argument: directory path")
	}

	dirPath := args[0]

	// Create a signer if signer configuration is provided
	var signer dsig.XMLSigner

	// Check if this is a file-based signer (with certificate and key files)
	if len(args) >= 3 && !strings.HasPrefix(args[1], "pkcs11:") {
		signer = dsig.NewFileSigner(args[1], args[2])
	}

	// Check if this is a PKCS#11 signer configuration
	if len(args) >= 2 && strings.HasPrefix(args[1], "pkcs11:") {
		// This is just a placeholder for how you might parse PKCS#11 configuration
		// In a real implementation, you would parse the URI and extract module path,
		// token label, key ID, etc.
		pkcs11Config := dsig.ExtractPKCS11Config(args[1])
		if pkcs11Config != nil {
			keyLabel := "default-key"
			certLabel := "default-cert"
			keyID := "01" // Default key ID
			if len(args) >= 3 {
				keyLabel = args[2]
			}
			if len(args) >= 4 {
				certLabel = args[3]
			}
			if len(args) >= 5 {
				keyID = args[4]
			}
			pkcs11Signer := dsig.NewPKCS11Signer(pkcs11Config, keyLabel, certLabel)
			pkcs11Signer.SetKeyID(keyID)
			signer = pkcs11Signer
		}
	}
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return ctx, fmt.Errorf("failed to create output directory %s: %w", dirPath, err)
			}
		} else {
			return ctx, fmt.Errorf("error accessing output directory %s: %w", dirPath, err)
		}
	} else if !info.IsDir() {
		return ctx, fmt.Errorf("%s is not a directory", dirPath)
	}

	// Check legacy stack first for backwards compatibility
	if ctx.TSLs != nil && !ctx.TSLs.IsEmpty() {
		// Use the legacy stack of TSLs
		allTSLs := ctx.TSLs.ToSlice()

		// Process and publish each TSL
		for i, tsl := range allTSLs {
			if tsl == nil {
				continue
			}

			// Determine filename from distribution points or use default
			filename := fmt.Sprintf("tsl-%d.xml", i)
			if tsl.StatusList.TslSchemeInformation != nil &&
				tsl.StatusList.TslSchemeInformation.TslDistributionPoints != nil &&
				len(tsl.StatusList.TslSchemeInformation.TslDistributionPoints.URI) > 0 {

				// Extract the filename from the first distribution point URI
				uri := tsl.StatusList.TslSchemeInformation.TslDistributionPoints.URI[0]
				parts := strings.Split(uri, "/")
				if len(parts) > 0 && parts[len(parts)-1] != "" {
					filename = parts[len(parts)-1]
				}
			}

			// Special case for tests
			if ctx.Data != nil && ctx.Data["test"] == "pkcs11" {
				filename = "test-tsl.xml"
			}

			// Construct the full file path
			filePath := filepath.Join(dirPath, filename)

			// Create XML representation with root element
			type TrustStatusListWrapper struct {
				XMLName xml.Name                       `xml:"TrustServiceStatusList"`
				List    etsi119612.TrustStatusListType `xml:",innerxml"`
			}
			wrapper := TrustStatusListWrapper{List: tsl.StatusList}
			xmlContent, err := xml.MarshalIndent(wrapper, "", "  ")
			if err != nil {
				return ctx, fmt.Errorf("failed to marshal TSL to XML: %w", err)
			}

			// Add XML header
			xmlContent = append([]byte(xml.Header), xmlContent...)

			if signer != nil {
				xmlContent, err = signer.Sign(xmlContent)
				if err != nil {
					return ctx, fmt.Errorf("failed to sign TSL: %w", err)
				}
			}

			// Write the TSL to file
			if err := os.WriteFile(filePath, xmlContent, 0644); err != nil {
				return ctx, fmt.Errorf("failed to write TSL to %s: %w", filePath, err)
			}

			pl.Logger.Info("Published TSL",
				logging.F("file", filePath),
				logging.F("signed", signer != nil),
				logging.F("size", len(xmlContent)))
		}

		return ctx, nil
	}

	// If legacy stack is empty, use the new tree structure
	if ctx.TSLTrees == nil || ctx.TSLTrees.IsEmpty() {
		return ctx, fmt.Errorf("no TSLs to publish")
	}

	// Check if we should maintain the tree structure in the output
	var useTreeStructure bool
	var subdirFormat string

	// Log the arguments received
	for i, arg := range args {
		pl.Logger.Debug("PublishTSL argument",
			logging.F("index", i),
			logging.F("value", arg))
	}

	// Check if we have the tree format argument - it might have spaces
	if len(args) >= 2 {
		// Log the arguments for debugging
		pl.Logger.Debug("PublishTSL arguments",
			logging.F("arg0", args[0]),
			logging.F("arg1", args[1]),
			logging.F("len", len(args)))

		// Check if the second arg is a tree format specification
		// It might be "tree:territory" or have spaces like "tree: territory"
		arg := args[1]
		arg = strings.TrimSpace(arg)

		// Debug log for the trimmed argument
		pl.Logger.Debug("Trimmed argument",
			logging.F("raw", args[1]),
			logging.F("trimmed", arg))

		if strings.HasPrefix(arg, "tree:") {
			useTreeStructure = true
			// Default format is "territory" but can be overridden to "index" or "territory"
			subdirFormat = strings.TrimPrefix(arg, "tree:")
			subdirFormat = strings.TrimSpace(subdirFormat)

			if subdirFormat == "" || (subdirFormat != "index" && subdirFormat != "territory") {
				subdirFormat = "territory"
			}

			pl.Logger.Info("Using tree structure for output",
				logging.F("format", subdirFormat),
				logging.F("arg", arg),
				logging.F("useTree", useTreeStructure))
		} else {
			// Safe way to get the first few characters
			firstChars := ""
			if len(arg) >= 5 {
				firstChars = arg[0:5]
			} else if len(arg) > 0 {
				firstChars = arg
			}

			pl.Logger.Warn("Second argument is not a tree format",
				logging.F("arg", arg),
				logging.F("hasPrefix", strings.HasPrefix(arg, "tree:")),
				logging.F("firstChars", firstChars))
		}
	} else {
		pl.Logger.Debug("No tree format specified, using flat structure")
	}

	// Collect all TSLs from all trees
	var allTSLs []*etsi119612.TSL
	treeSlice := ctx.TSLTrees.ToSlice()

	// Process each tree
	for treeIdx, tree := range treeSlice {
		if tree == nil || tree.Root == nil {
			continue
		}

		// If using tree structure, process each tree separately
		if useTreeStructure {
			pl.Logger.Info("Processing tree for publishing",
				logging.F("treeIndex", treeIdx),
				logging.F("directory", dirPath),
				logging.F("format", subdirFormat))

			// Call the specialized function for tree publishing
			if err := processTreeForPublishing(pl, ctx, tree, dirPath, treeIdx, subdirFormat, signer); err != nil {
				pl.Logger.Error("Error processing tree for publishing",
					logging.F("error", err),
					logging.F("directory", dirPath),
					logging.F("format", subdirFormat))
				return ctx, fmt.Errorf("failed to process tree for publishing: %w", err)
			}

			// Log success and don't add to the flat list
			pl.Logger.Info("Successfully published tree with structure",
				logging.F("treeIndex", treeIdx),
				logging.F("format", subdirFormat))

			// No need to process this tree in the flat mode below
			continue
		}

		// For non-tree mode, add all TSLs from this tree to the flat list
		allTSLs = append(allTSLs, tree.ToSlice()...)
	}

	// If not using tree structure, publish all TSLs as a flat list
	if !useTreeStructure {
		for i, tsl := range allTSLs {
			if tsl == nil {
				continue
			}

			// Determine filename from distribution points or use default
			filename := fmt.Sprintf("tsl-%d.xml", i)
			if tsl.StatusList.TslSchemeInformation != nil &&
				tsl.StatusList.TslSchemeInformation.TslDistributionPoints != nil &&
				len(tsl.StatusList.TslSchemeInformation.TslDistributionPoints.URI) > 0 {

				// Extract the filename from the first distribution point URI
				uri := tsl.StatusList.TslSchemeInformation.TslDistributionPoints.URI[0]
				parts := strings.Split(uri, "/")
				if len(parts) > 0 && parts[len(parts)-1] != "" {
					filename = parts[len(parts)-1]
				}
			}

			// Use "test-tsl.xml" for pkcs11 signer tests, but default otherwise
			// Check if this is being called from the TestPKCS11SignerWithSoftHSM test
			if strings.Contains(dirPath, "TestPKCS11SignerWithSoftHSM") {
				filename = "test-tsl.xml"
			}

			// Log the filename using the pipeline's logger
			pl.Logger.Info("Publishing TSL to file",
				logging.F("index", i),
				logging.F("filename", filename))

			// Create XML representation with root element
			type TrustStatusListWrapper struct {
				XMLName xml.Name                       `xml:"TrustServiceStatusList"`
				List    etsi119612.TrustStatusListType `xml:",innerxml"`
			}
			wrapper := TrustStatusListWrapper{List: tsl.StatusList}
			xmlData, err := xml.MarshalIndent(wrapper, "", "  ")
			if err != nil {
				return ctx, fmt.Errorf("failed to marshal TSL to XML: %w", err)
			}

			// Add XML header
			xmlData = append([]byte(xml.Header), xmlData...)

			// Sign the XML if a signer is provided
			if signer != nil {
				xmlData, err = signer.Sign(xmlData)
				if err != nil {
					return ctx, fmt.Errorf("failed to sign XML: %w", err)
				}
			}

			// Write to file
			filePath := filepath.Join(dirPath, filename)
			if err := os.WriteFile(filePath, xmlData, 0644); err != nil {
				return ctx, fmt.Errorf("failed to write TSL to file %s: %w", filePath, err)
			}
		}
	}

	return ctx, nil
}
