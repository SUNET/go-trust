package pipeline

import (
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"	// Parse the XML into a TSL
	var tslObj etsi119612.TSL
	err := xml.Unmarshal([]byte(tslXML), &tslObj)
	require.NoError(t, err)
	
	// Create a context with the TSL
	ctx := NewContext()
	ctx.TSLs.Push(&tslObj)ub.com/SUNET/g119612/pkg/etsi119612"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformTSL(t *testing.T) {
	// Skip if xsltproc is not available
	if _, err := exec.LookPath("xsltproc"); err != nil {
		t.Skip("xsltproc not available, skipping test")
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "tsl-transform-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a simple XSLT stylesheet
	xsltPath := filepath.Join(tempDir, "transform.xslt")
	xsltContent := `<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
                xmlns:tsl="http://uri.etsi.org/02231/v2#">
  <xsl:output method="xml" indent="yes"/>
  
  <!-- Identity transform -->
  <xsl:template match="@*|node()">
    <xsl:copy>
      <xsl:apply-templates select="@*|node()"/>
    </xsl:copy>
  </xsl:template>
  
  <!-- Add a test attribute to the root element -->
  <xsl:template match="/*">
    <xsl:copy>
      <xsl:attribute name="testAttribute">transformed</xsl:attribute>
      <xsl:apply-templates select="@*|node()"/>
    </xsl:copy>
  </xsl:template>
</xsl:stylesheet>`
	
	err = os.WriteFile(xsltPath, []byte(xsltContent), 0644)
	require.NoError(t, err)

	// Create output directory
	outputDir := filepath.Join(tempDir, "output")
	
	// Create a simple TSL for testing
	// We need to use a pre-made TSL XML document to avoid complex struct creation
	tslXML := `	// Create a simple TSL with minimal data for testing
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{},
	}

	// Create a temporary XML file for the test
	tempTSLFile := filepath.Join(tempDir, "test-tsl.xml")
	tslXML := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
  <SchemeInformation>
    <TSLVersionIdentifier>5</TSLVersionIdentifier>
    <TSLSequenceNumber>1</TSLSequenceNumber>
    <TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
    <DistributionPoints>
      <URI>http://example.com/tsl/test-tsl.xml</URI>
    </DistributionPoints>
  </SchemeInformation>
</TrustServiceStatusList>`
	
	err = os.WriteFile(tempTSLFile, []byte(tslXML), 0644)
	require.NoError(t, err)
	
	// Create a context with the TSL
	ctx := NewContext()
	ctx.TSLs.Push(tsl)`

	// Parse the XML into a TSL
	var tslObj etsi119612.TSL
	err = xml.Unmarshal([]byte(tslXML), &tslObj)
	require.NoError(t, err)
	
	// Create a context with the TSL
	ctx := NewContext()
	ctx.TSLs.Push(&tslObj)

	// Setup context
	ctx := NewContext()
	ctx.TSLs.Push(tsl)

	t.Run("Transform and Replace", func(t *testing.T) {
		// Call the TransformTSL function with replace mode
		resultCtx, err := TransformTSL(nil, ctx, xsltPath, "replace")
		assert.NoError(t, err)
		assert.NotNil(t, resultCtx)
		assert.Equal(t, 1, resultCtx.TSLs.Size())
		
		// Get the transformed TSL
		transformedTSLs := resultCtx.TSLs.ToSlice()
		assert.Len(t, transformedTSLs, 1)
		
		// To verify the transformation, we'll marshal to XML and check for the attribute
		// in our transform_test.go step function directly
	})

	t.Run("Transform and Output to Directory", func(t *testing.T) {
		// Call the TransformTSL function with output directory
		resultCtx, err := TransformTSL(nil, ctx, xsltPath, outputDir)
		assert.NoError(t, err)
		assert.NotNil(t, resultCtx)
		
		// Check that the file was created
		expectedFile := filepath.Join(outputDir, "test-tsl.xml")
		_, err = os.Stat(expectedFile)
		assert.NoError(t, err, "Output file should exist")
		
		// Read the file content
		content, err := os.ReadFile(expectedFile)
		assert.NoError(t, err)
		
		// Check if the content contains the transformation
		assert.True(t, strings.Contains(string(content), `testAttribute="transformed"`))
	})

	t.Run("Error Cases", func(t *testing.T) {
		// Test missing arguments
		_, err := TransformTSL(nil, ctx)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "missing required arguments"))
		
		// Test non-existent stylesheet
		_, err = TransformTSL(nil, ctx, "/nonexistent/path.xslt", "replace")
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "XSLT stylesheet not found"))
		
		// Test empty context
		emptyCtx := NewContext()
		emptyCtx.TSLs = nil
		_, err = TransformTSL(nil, emptyCtx, xsltPath, "replace")
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "no TSLs to transform"))
	})
}