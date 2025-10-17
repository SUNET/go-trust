package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test server defaults
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Default host = %v, want %v", cfg.Server.Host, "127.0.0.1")
	}
	if cfg.Server.Port != "6001" {
		t.Errorf("Default port = %v, want %v", cfg.Server.Port, "6001")
	}
	if cfg.Server.Frequency != 5*time.Minute {
		t.Errorf("Default frequency = %v, want %v", cfg.Server.Frequency, 5*time.Minute)
	}

	// Test logging defaults
	if cfg.Logging.Level != "info" {
		t.Errorf("Default log level = %v, want %v", cfg.Logging.Level, "info")
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("Default log format = %v, want %v", cfg.Logging.Format, "text")
	}
	if cfg.Logging.Output != "stdout" {
		t.Errorf("Default log output = %v, want %v", cfg.Logging.Output, "stdout")
	}

	// Test pipeline defaults
	if cfg.Pipeline.Timeout != 30*time.Second {
		t.Errorf("Default timeout = %v, want %v", cfg.Pipeline.Timeout, 30*time.Second)
	}
	if cfg.Pipeline.MaxRequestSize != 10*1024*1024 {
		t.Errorf("Default max request size = %v, want %v", cfg.Pipeline.MaxRequestSize, 10*1024*1024)
	}
	if cfg.Pipeline.MaxRedirects != 3 {
		t.Errorf("Default max redirects = %v, want %v", cfg.Pipeline.MaxRedirects, 3)
	}

	// Test security defaults
	if cfg.Security.RateLimitRPS != 100 {
		t.Errorf("Default rate limit = %v, want %v", cfg.Security.RateLimitRPS, 100)
	}
	if cfg.Security.EnableCORS {
		t.Error("Default CORS should be disabled")
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  host: "0.0.0.0"
  port: "8080"
  frequency: "10m"

logging:
  level: "debug"
  format: "json"
  output: "/var/log/go-trust.log"

pipeline:
  timeout: "60s"
  max_request_size: 20971520
  max_redirects: 5
  allowed_hosts:
    - "*.europa.eu"
    - "*.example.com"

security:
  rate_limit_rps: 200
  enable_cors: true
  allowed_origins:
    - "https://example.com"
    - "https://test.com"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify server configuration
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Host = %v, want %v", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Server.Port != "8080" {
		t.Errorf("Port = %v, want %v", cfg.Server.Port, "8080")
	}
	if cfg.Server.Frequency != 10*time.Minute {
		t.Errorf("Frequency = %v, want %v", cfg.Server.Frequency, 10*time.Minute)
	}

	// Verify logging configuration
	if cfg.Logging.Level != "debug" {
		t.Errorf("Log level = %v, want %v", cfg.Logging.Level, "debug")
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("Log format = %v, want %v", cfg.Logging.Format, "json")
	}
	if cfg.Logging.Output != "/var/log/go-trust.log" {
		t.Errorf("Log output = %v, want %v", cfg.Logging.Output, "/var/log/go-trust.log")
	}

	// Verify pipeline configuration
	if cfg.Pipeline.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Pipeline.Timeout, 60*time.Second)
	}
	if cfg.Pipeline.MaxRequestSize != 20971520 {
		t.Errorf("Max request size = %v, want %v", cfg.Pipeline.MaxRequestSize, 20971520)
	}
	if cfg.Pipeline.MaxRedirects != 5 {
		t.Errorf("Max redirects = %v, want %v", cfg.Pipeline.MaxRedirects, 5)
	}
	if len(cfg.Pipeline.AllowedHosts) != 2 {
		t.Errorf("Allowed hosts count = %v, want %v", len(cfg.Pipeline.AllowedHosts), 2)
	}

	// Verify security configuration
	if cfg.Security.RateLimitRPS != 200 {
		t.Errorf("Rate limit RPS = %v, want %v", cfg.Security.RateLimitRPS, 200)
	}
	if !cfg.Security.EnableCORS {
		t.Error("CORS should be enabled")
	}
	if len(cfg.Security.AllowedOrigins) != 2 {
		t.Errorf("Allowed origins count = %v, want %v", len(cfg.Security.AllowedOrigins), 2)
	}
}

func TestLoadConfigWithEnvOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("GT_HOST", "192.168.1.1")
	os.Setenv("GT_PORT", "9000")
	os.Setenv("GT_FREQUENCY", "15m")
	os.Setenv("GT_LOG_LEVEL", "warn")
	os.Setenv("GT_LOG_FORMAT", "json")
	os.Setenv("GT_LOG_OUTPUT", "stderr")
	os.Setenv("GT_RATE_LIMIT_RPS", "500")
	os.Setenv("GT_ENABLE_CORS", "true")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("GT_HOST")
		os.Unsetenv("GT_PORT")
		os.Unsetenv("GT_FREQUENCY")
		os.Unsetenv("GT_LOG_LEVEL")
		os.Unsetenv("GT_LOG_FORMAT")
		os.Unsetenv("GT_LOG_OUTPUT")
		os.Unsetenv("GT_RATE_LIMIT_RPS")
		os.Unsetenv("GT_ENABLE_CORS")
	}()

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify environment variables were applied
	if cfg.Server.Host != "192.168.1.1" {
		t.Errorf("Host = %v, want %v", cfg.Server.Host, "192.168.1.1")
	}
	if cfg.Server.Port != "9000" {
		t.Errorf("Port = %v, want %v", cfg.Server.Port, "9000")
	}
	if cfg.Server.Frequency != 15*time.Minute {
		t.Errorf("Frequency = %v, want %v", cfg.Server.Frequency, 15*time.Minute)
	}
	if cfg.Logging.Level != "warn" {
		t.Errorf("Log level = %v, want %v", cfg.Logging.Level, "warn")
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("Log format = %v, want %v", cfg.Logging.Format, "json")
	}
	if cfg.Logging.Output != "stderr" {
		t.Errorf("Log output = %v, want %v", cfg.Logging.Output, "stderr")
	}
	if cfg.Security.RateLimitRPS != 500 {
		t.Errorf("Rate limit RPS = %v, want %v", cfg.Security.RateLimitRPS, 500)
	}
	if !cfg.Security.EnableCORS {
		t.Error("CORS should be enabled")
	}
}

func TestLoadConfigInvalidFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("LoadConfig() should fail with nonexistent file")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	if err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig() should fail with invalid YAML")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "Valid default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "Empty port",
			config: &Config{
				Server:   ServerConfig{Host: "127.0.0.1", Port: "", Frequency: 5 * time.Minute},
				Logging:  LoggingConfig{Level: "info", Format: "text", Output: "stdout"},
				Pipeline: PipelineConfig{Timeout: 30 * time.Second, MaxRequestSize: 1024, MaxRedirects: 3},
				Security: SecurityConfig{RateLimitRPS: 100},
			},
			wantErr: true,
		},
		{
			name: "Negative frequency",
			config: &Config{
				Server:   ServerConfig{Host: "127.0.0.1", Port: "6001", Frequency: -1 * time.Minute},
				Logging:  LoggingConfig{Level: "info", Format: "text", Output: "stdout"},
				Pipeline: PipelineConfig{Timeout: 30 * time.Second, MaxRequestSize: 1024, MaxRedirects: 3},
				Security: SecurityConfig{RateLimitRPS: 100},
			},
			wantErr: true,
		},
		{
			name: "Invalid log level",
			config: &Config{
				Server:   ServerConfig{Host: "127.0.0.1", Port: "6001", Frequency: 5 * time.Minute},
				Logging:  LoggingConfig{Level: "invalid", Format: "text", Output: "stdout"},
				Pipeline: PipelineConfig{Timeout: 30 * time.Second, MaxRequestSize: 1024, MaxRedirects: 3},
				Security: SecurityConfig{RateLimitRPS: 100},
			},
			wantErr: true,
		},
		{
			name: "Invalid log format",
			config: &Config{
				Server:   ServerConfig{Host: "127.0.0.1", Port: "6001", Frequency: 5 * time.Minute},
				Logging:  LoggingConfig{Level: "info", Format: "invalid", Output: "stdout"},
				Pipeline: PipelineConfig{Timeout: 30 * time.Second, MaxRequestSize: 1024, MaxRedirects: 3},
				Security: SecurityConfig{RateLimitRPS: 100},
			},
			wantErr: true,
		},
		{
			name: "Negative timeout",
			config: &Config{
				Server:   ServerConfig{Host: "127.0.0.1", Port: "6001", Frequency: 5 * time.Minute},
				Logging:  LoggingConfig{Level: "info", Format: "text", Output: "stdout"},
				Pipeline: PipelineConfig{Timeout: -1 * time.Second, MaxRequestSize: 1024, MaxRedirects: 3},
				Security: SecurityConfig{RateLimitRPS: 100},
			},
			wantErr: true,
		},
		{
			name: "Negative max request size",
			config: &Config{
				Server:   ServerConfig{Host: "127.0.0.1", Port: "6001", Frequency: 5 * time.Minute},
				Logging:  LoggingConfig{Level: "info", Format: "text", Output: "stdout"},
				Pipeline: PipelineConfig{Timeout: 30 * time.Second, MaxRequestSize: -1, MaxRedirects: 3},
				Security: SecurityConfig{RateLimitRPS: 100},
			},
			wantErr: true,
		},
		{
			name: "Negative max redirects",
			config: &Config{
				Server:   ServerConfig{Host: "127.0.0.1", Port: "6001", Frequency: 5 * time.Minute},
				Logging:  LoggingConfig{Level: "info", Format: "text", Output: "stdout"},
				Pipeline: PipelineConfig{Timeout: 30 * time.Second, MaxRequestSize: 1024, MaxRedirects: -1},
				Security: SecurityConfig{RateLimitRPS: 100},
			},
			wantErr: true,
		},
		{
			name: "Non-positive rate limit",
			config: &Config{
				Server:   ServerConfig{Host: "127.0.0.1", Port: "6001", Frequency: 5 * time.Minute},
				Logging:  LoggingConfig{Level: "info", Format: "text", Output: "stdout"},
				Pipeline: PipelineConfig{Timeout: 30 * time.Second, MaxRequestSize: 1024, MaxRedirects: 3},
				Security: SecurityConfig{RateLimitRPS: 0},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvOverridesWithPipelineAndSecurityConfig(t *testing.T) {
	// Set additional environment variables
	os.Setenv("GT_PIPELINE_TIMEOUT", "120s")
	os.Setenv("GT_MAX_REQUEST_SIZE", "52428800")
	os.Setenv("GT_MAX_REDIRECTS", "10")
	os.Setenv("GT_ALLOWED_HOSTS", "*.example.com,*.test.org")
	os.Setenv("GT_ALLOWED_ORIGINS", "https://app1.com,https://app2.com")

	defer func() {
		os.Unsetenv("GT_PIPELINE_TIMEOUT")
		os.Unsetenv("GT_MAX_REQUEST_SIZE")
		os.Unsetenv("GT_MAX_REDIRECTS")
		os.Unsetenv("GT_ALLOWED_HOSTS")
		os.Unsetenv("GT_ALLOWED_ORIGINS")
	}()

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify pipeline environment variables
	if cfg.Pipeline.Timeout != 120*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Pipeline.Timeout, 120*time.Second)
	}
	if cfg.Pipeline.MaxRequestSize != 52428800 {
		t.Errorf("Max request size = %v, want %v", cfg.Pipeline.MaxRequestSize, 52428800)
	}
	if cfg.Pipeline.MaxRedirects != 10 {
		t.Errorf("Max redirects = %v, want %v", cfg.Pipeline.MaxRedirects, 10)
	}
	if len(cfg.Pipeline.AllowedHosts) != 2 {
		t.Errorf("Allowed hosts count = %v, want %v", len(cfg.Pipeline.AllowedHosts), 2)
	}

	// Verify security environment variables
	if len(cfg.Security.AllowedOrigins) != 2 {
		t.Errorf("Allowed origins count = %v, want %v", len(cfg.Security.AllowedOrigins), 2)
	}
}
