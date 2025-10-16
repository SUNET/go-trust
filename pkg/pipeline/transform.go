// Package pipeline provides a pipeline framework for processing Trust Status Lists (TSLs).
package pipeline

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/SUNET/g119612/pkg/etsi119612"
	"github.com/SUNET/go-trust/xslt"
)

// TransformTSL applies an XSLT transformation to each TSL in the context.
// This pipeline step allows for flexible transformation of TSL XML documents
// using XSLT stylesheets. It can either replace the TSLs in the pipeline context
// with their transformed versions, or output the transformed documents to a
// specified directory.
//
// The step requires the 'xsltproc' command to be available on the system.
//
// Arguments:
//   - arg[0]: Path to the XSLT stylesheet. Can be a filesystem path or an embedded XSLT path.
//     For embedded XSLTs, use the format 'embedded:filename.xslt'.
//     e.g., 'embedded:tsl-to-html.xslt' for the embedded TSL-to-HTML stylesheet.
//   - arg[1]: Mode: "replace" or directory path.
//   - If "replace", transformed TSLs replace the originals in the context.
//   - Otherwise, it's treated as a directory path where transformed TSLs are saved.
//   - arg[2]: (Optional) Output file extension (default: "xml")
//
// Example usage in pipeline YAML for file-based XSLT:
//
//   - transform:
//   - /path/to/stylesheet.xslt
//   - replace
//
// OR for embedded XSLT:
//
//   - transform:
//   - embedded:tsl-to-html.xslt
//   - /output/directory
//   - html
func TransformTSL(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 2 {
		return ctx, fmt.Errorf("missing required arguments: need XSLT stylesheet path and mode ('replace' or output directory)")
	}

	// Parse arguments
	xsltPath := args[0]
	mode := args[1]
	extension := "xml"
	if len(args) >= 3 {
		extension = args[2]
	}

	// Check if this is an embedded XSLT or a file path
	isEmbedded := xslt.IsEmbeddedPath(xsltPath)

	// Check if the XSLT file exists (if it's not embedded)
	if !isEmbedded {
		if _, err := os.Stat(xsltPath); os.IsNotExist(err) {
			return ctx, fmt.Errorf("XSLT stylesheet not found at path: %s", xsltPath)
		}
	}

	// Check if we need to create an output directory
	isReplace := mode == "replace"
	var outputDir string
	if !isReplace {
		outputDir = mode
		// Create output directory if it doesn't exist
		info, err := os.Stat(outputDir)
		if err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return ctx, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
				}
			} else {
				return ctx, fmt.Errorf("error accessing output directory %s: %w", outputDir, err)
			}
		} else if !info.IsDir() {
			return ctx, fmt.Errorf("%s is not a directory", outputDir)
		}
	}

	if ctx.TSLTrees == nil || ctx.TSLTrees.IsEmpty() {
		return ctx, fmt.Errorf("no TSLs to transform")
	}

	// Collect all TSLs from all trees
	var allTSLs []*etsi119612.TSL
	treeSlice := ctx.TSLTrees.ToSlice()
	for _, tree := range treeSlice {
		if tree == nil {
			continue
		}
		allTSLs = append(allTSLs, tree.ToSlice()...)
	}

	// Setup for transformations
	transformedTSLs := make([]*etsi119612.TSL, 0, len(allTSLs))
	tsls := allTSLs

	for i, tsl := range tsls {
		if tsl == nil {
			continue
		}

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

		// Apply XSLT transformation
		var transformedXML []byte
		if isEmbedded {
			// Use embedded XSLT
			embeddedName := xslt.ExtractNameFromPath(xsltPath)
			transformedXML, err = applyEmbeddedXSLTTransformation(xmlData, embeddedName)
		} else {
			// Use external XSLT file
			transformedXML, err = applyFileXSLTTransformation(xmlData, xsltPath)
		}

		if err != nil {
			return ctx, fmt.Errorf("XSLT transformation failed for TSL %d: %w", i, err)
		}

		if isReplace {
			// Parse the transformed XML back into a TSL structure
			var transformedWrapper TrustStatusListWrapper
			if err := xml.Unmarshal(transformedXML, &transformedWrapper); err != nil {
				return ctx, fmt.Errorf("failed to parse transformed XML for TSL %d: %w", i, err)
			}

			// Create a new TSL with the transformed content
			transformedTSL := &etsi119612.TSL{
				StatusList: transformedWrapper.List,
			}
			transformedTSLs = append(transformedTSLs, transformedTSL)
		} else {
			// Determine filename for output
			filename := fmt.Sprintf("transformed-tsl-%d.%s", i, extension)
			if tsl.StatusList.TslSchemeInformation != nil &&
				tsl.StatusList.TslSchemeInformation.TslDistributionPoints != nil &&
				len(tsl.StatusList.TslSchemeInformation.TslDistributionPoints.URI) > 0 {

				// Extract the filename from the first distribution point URI
				uri := tsl.StatusList.TslSchemeInformation.TslDistributionPoints.URI[0]
				parts := strings.Split(uri, "/")
				if len(parts) > 0 && parts[len(parts)-1] != "" {
					baseName := parts[len(parts)-1]
					filename = fmt.Sprintf("%s.%s", strings.TrimSuffix(baseName, filepath.Ext(baseName)), extension)
				}
			}

			// Write transformed XML to file
			filePath := filepath.Join(outputDir, filename)
			if err := os.WriteFile(filePath, transformedXML, 0644); err != nil {
				return ctx, fmt.Errorf("failed to write transformed TSL to file %s: %w", filePath, err)
			}
		}
	}

	// Replace the TSLs in the context if in replace mode
	if isReplace {
		// Clear the existing tree stack
		ctx.TSLTrees = nil
		ctx.EnsureTSLTrees()

		// Add each transformed TSL as a new tree
		for _, transformedTSL := range transformedTSLs {
			tree := NewTSLTree(transformedTSL)
			ctx.AddTSLTree(tree)
		}
	}

	return ctx, nil
}

// applyFileXSLTTransformation applies an XSLT transformation to XML data using an external XSLT file
func applyFileXSLTTransformation(xmlData []byte, xsltPath string) ([]byte, error) {
	// Create a temporary file for the input XML
	tempFile, err := os.CreateTemp("", "input-*.xml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	// Write XML data to the temp file
	if _, err := tempFile.Write(xmlData); err != nil {
		return nil, fmt.Errorf("failed to write XML to temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Run xsltproc command to apply the transformation
	cmd := exec.Command("xsltproc", xsltPath, tempFile.Name())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("xsltproc error: %w - %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// applyEmbeddedXSLTTransformation applies an XSLT transformation to XML data using an embedded XSLT file
func applyEmbeddedXSLTTransformation(xmlData []byte, xsltName string) ([]byte, error) {
	// Get the embedded XSLT content
	xsltContent, err := xslt.Get(xsltName)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedded XSLT: %w", err)
	}

	// Create a temporary file for the input XML
	tempXmlFile, err := os.CreateTemp("", "input-*.xml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp XML file: %w", err)
	}
	defer os.Remove(tempXmlFile.Name())

	// Write XML data to the temp file
	if _, err := tempXmlFile.Write(xmlData); err != nil {
		return nil, fmt.Errorf("failed to write XML to temp file: %w", err)
	}
	if err := tempXmlFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp XML file: %w", err)
	}

	// Create a temporary file for the XSLT
	tempXsltFile, err := os.CreateTemp("", "style-*.xslt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp XSLT file: %w", err)
	}
	defer os.Remove(tempXsltFile.Name())

	// Write XSLT data to the temp file
	if _, err := tempXsltFile.Write(xsltContent); err != nil {
		return nil, fmt.Errorf("failed to write XSLT to temp file: %w", err)
	}
	if err := tempXsltFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp XSLT file: %w", err)
	}

	// Run xsltproc command to apply the transformation
	cmd := exec.Command("xsltproc", tempXsltFile.Name(), tempXmlFile.Name())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("xsltproc error: %w - %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

func init() {
	// Register the TransformTSL function
	RegisterFunction("transform", TransformTSL)
}
