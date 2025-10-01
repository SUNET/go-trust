package pipeline

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"text/template"

	"github.com/SUNET/g119612/pkg/etsi119612"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

var testCertBase64 string
var testCertDER []byte
var testCert *x509.Certificate

// generateTestCertBase64 runs openssl to generate a self-signed cert and returns the base64-encoded DER string.
func generateTestCertBase64() (string, []byte, *x509.Certificate, error) {
	keyFile, err := os.CreateTemp("", "testkey-*.pem")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp key file: %w", err)
	}
	defer os.Remove(keyFile.Name())
	keyFile.Close()
	certFile, err := os.CreateTemp("", "testcert-*.pem")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp cert file: %w", err)
	}
	defer os.Remove(certFile.Name())
	certFile.Close()
	derFile, err := os.CreateTemp("", "testcert-*.der")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp der file: %w", err)
	}
	defer os.Remove(derFile.Name())
	derFile.Close()

	opensslCmd := fmt.Sprintf("openssl req -x509 -newkey rsa:2048 -keyout %s -out %s -days 365 -nodes -subj '/CN=Test Cert' 2>/dev/null && openssl x509 -outform der -in %s -out %s 2>/dev/null && openssl base64 -in %s -A 2>/dev/null", keyFile.Name(), certFile.Name(), certFile.Name(), derFile.Name(), derFile.Name())
	cmd := exec.Command("bash", "-c", opensslCmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	output := out.String()
	if err != nil {
		return "", nil, nil, fmt.Errorf("openssl error: %v\noutput: %s", err, output)
	}
	certBase64 := strings.TrimSpace(output)
	certDER, err := base64.StdEncoding.DecodeString(certBase64)
	if err != nil {
		return certBase64, nil, nil, fmt.Errorf("base64 decode error: %v\noutput: %s", err, output)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return certBase64, certDER, nil, fmt.Errorf("parse cert error: %v\noutput: %s", err, output)
	}
	return certBase64, certDER, cert, nil
}

func init() {
	var err error
	testCertBase64, testCertDER, testCert, err = generateTestCertBase64()
	if err != nil {
		panic("failed to generate test cert: " + err.Error())
	}
}

func TestPipeline_Process_Success(t *testing.T) {
	RegisterFunction("testfunc", func(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
		assert.Equal(t, []string{"foo", "bar"}, args)
		if ctx == nil {
			t.Fatal("ctx should not be nil")
		}
		ctx.TSLs = append(ctx.TSLs, nil) // simulate adding a TSL
		return ctx, nil
	})
	yamlData := `
- testfunc:
    - foo
    - bar
`
	var pipes []Pipe
	err := yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.NoError(t, err)
	pl := &Pipeline{Pipes: pipes}
	ctx, err := pl.Process(&Context{})
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
	assert.Len(t, ctx.TSLs, 1)
}

func TestPipeline_Process_UnknownMethod(t *testing.T) {
	yamlData := `
- unknown:
    - foo
`
	var pipes []Pipe
	err := yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.NoError(t, err)
	pl := &Pipeline{Pipes: pipes}
	ctx, err := pl.Process(&Context{})
	assert.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "unknown methodName")
}

func TestPipeline_Process_FuncError(t *testing.T) {
	RegisterFunction("failfunc", func(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
		return ctx, os.ErrPermission
	})
	yamlData := `
- failfunc:
    - foo
`
	var pipes []Pipe
	err := yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.NoError(t, err)
	pl := &Pipeline{Pipes: pipes}
	ctx, err := pl.Process(&Context{})
	assert.Error(t, err)
	assert.NotNil(t, ctx)
	assert.Contains(t, err.Error(), "failed")
}

// TestPipeline_SelectStep tests the select pipeline step with a local test TSL XML file.
func TestPipeline_SelectStep(t *testing.T) {
	// Render the XML template with the generated test certificate
	tmplBytes, err := os.ReadFile("./testdata/test-tsl.xml")
	assert.NoError(t, err)
	tmpl, err := template.New("tsl").Parse(string(tmplBytes))
	assert.NoError(t, err)
	tmpfile, err := os.CreateTemp("", "test-tsl-*.xml")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	err = tmpl.Execute(tmpfile, map[string]string{"X509Certificate": testCertBase64})
	assert.NoError(t, err)
	tmpfile.Close()
	yamlData := `
- load: ["file://` + tmpfile.Name() + `"]
- select: []
`
	var pipes []Pipe
	err = yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.NoError(t, err)
	pl := &Pipeline{Pipes: pipes}
	ctx, err := pl.Process(&Context{})
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.CertPool)
	if ctx.CertPool != nil {
		opts := x509.VerifyOptions{
			Roots: ctx.CertPool,
		}
		_, err := testCert.Verify(opts)
		assert.NoError(t, err, "testCert should verify against the CertPool")
	}
}

func TestPipeline_EchoStep(t *testing.T) {
	initialCtx := &Context{TSLs: []*etsi119612.TSL{nil}}
	pipes := []Pipe{{MethodName: "echo", MethodArguments: []string{"foo", "bar"}}}
	pl := &Pipeline{Pipes: pipes}
	ctx, err := pl.Process(initialCtx)
	if err != nil {
		t.Fatalf("echo step failed: %v", err)
	}
	if ctx != initialCtx {
		t.Errorf("echo step should return the same context instance")
	}
}

func TestNewPipeline(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "pipeline-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	yamlContent := `
- echo: ["foo", "bar"]
`
	if _, err := tmpfile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	pl, err := NewPipeline(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewPipeline failed: %v", err)
	}
	if len(pl.Pipes) != 1 || pl.Pipes[0].MethodName != "echo" {
		t.Errorf("Expected one echo step, got: %+v", pl.Pipes)
	}

	// Test error case: file does not exist
	_, err = NewPipeline("/nonexistent/file.yaml")
	if err == nil {
		t.Errorf("Expected error for nonexistent file, got nil")
	}
}

func TestSelectCertPool_EdgeCases(t *testing.T) {
	// No TSLs
	ctx := &Context{TSLs: nil}
	_, err := selectCertPool(nil, ctx)
	if err == nil || err.Error() != "select: no TSLs loaded in context" {
		t.Errorf("Expected error for no TSLs, got: %v", err)
	}

	// TSLs with no matching policy (simulate with empty TSLs)
	tsls := []*etsi119612.TSL{&etsi119612.TSL{}}
	ctx = &Context{TSLs: tsls}
	ctx, err = selectCertPool(nil, ctx, "nonexistent-policy")
	if err != nil {
		t.Errorf("Expected no error for no matching policy, got: %v", err)
	}
	if ctx.CertPool == nil {
		t.Errorf("Expected CertPool to be set for no matching policy")
	}

	// Multiple TSLs (simulate with two empty TSLs)
	tsls = []*etsi119612.TSL{&etsi119612.TSL{}, &etsi119612.TSL{}}
	ctx = &Context{TSLs: tsls}
	ctx, err = selectCertPool(nil, ctx)
	if err != nil {
		t.Errorf("Expected no error for multiple TSLs, got: %v", err)
	}
	if ctx.CertPool == nil {
		t.Errorf("Expected CertPool to be set for multiple TSLs")
	}
}

func TestLoadTSL_Errors(t *testing.T) {
	ctx := &Context{}
	// Invalid file path
	_, err := loadTSL(nil, ctx, "file:///nonexistent/path.xml")
	if err == nil {
		t.Errorf("Expected error for invalid file path, got nil")
	}

	// Invalid URL (unsupported scheme)
	_, err = loadTSL(nil, ctx, "ftp://example.com/tsl.xml")
	if err == nil {
		t.Errorf("Expected error for unsupported URL scheme, got nil")
	}

	// Invalid XML file
	tmpfile, err := os.CreateTemp("", "badxml-*.xml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte("not xml at all")); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpfile.Close()
	_, err = loadTSL(nil, ctx, "file://"+tmpfile.Name())
	if err == nil {
		t.Errorf("Expected error for invalid XML, got nil")
	}
}

func TestPipe_UnmarshalYAML_Errors(t *testing.T) {
	// Not a mapping node
	var pipes []Pipe
	yamlData := `- not-a-map`
	err := yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Pipe must be a map")

	// Mapping node with wrong structure (not a sequence for args)
	yamlData = `- testfunc: foo`
	err = yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Pipe arguments must be a sequence")
}
