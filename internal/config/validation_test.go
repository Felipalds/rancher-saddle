package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPath(t *testing.T) {
	// Get home directory for testing
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name        string
		input       string
		wantPrefix  string
		wantErr     bool
		description string
	}{
		{
			name:        "Empty path",
			input:       "",
			wantPrefix:  "",
			wantErr:     false,
			description: "Empty path should return empty string",
		},
		{
			name:        "Tilde only",
			input:       "~",
			wantPrefix:  homeDir,
			wantErr:     false,
			description: "~ should expand to home directory",
		},
		{
			name:        "Tilde with path",
			input:       "~/Downloads/test.pem",
			wantPrefix:  filepath.Join(homeDir, "Downloads", "test.pem"),
			wantErr:     false,
			description: "~/path should expand to home/path",
		},
		{
			name:        "Absolute path",
			input:       "/tmp/test.pem",
			wantPrefix:  "/tmp/test.pem",
			wantErr:     false,
			description: "Absolute paths should remain unchanged",
		},
		{
			name:        "Relative path",
			input:       "test.pem",
			wantPrefix:  "", // Will be current directory + test.pem
			wantErr:     false,
			description: "Relative paths should be converted to absolute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandPath(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// For empty input, check exact match
			if tt.input == "" {
				if got != tt.wantPrefix {
					t.Errorf("ExpandPath() = %v, want %v", got, tt.wantPrefix)
				}
				return
			}

			// For relative paths, just check it's absolute
			if tt.name == "Relative path" {
				if !filepath.IsAbs(got) {
					t.Errorf("ExpandPath() = %v, expected absolute path", got)
				}
				return
			}

			// For other cases, check the result matches expected
			if got != tt.wantPrefix {
				t.Errorf("ExpandPath() = %v, want %v (description: %s)", got, tt.wantPrefix, tt.description)
			}
		})
	}
}

func TestExpandPath_TildeExpansion(t *testing.T) {
	// Create a test to specifically verify tilde expansion
	testPath := "~/test/path"
	expanded, err := ExpandPath(testPath)

	if err != nil {
		t.Fatalf("ExpandPath() unexpected error: %v", err)
	}

	if strings.HasPrefix(expanded, "~") {
		t.Errorf("ExpandPath() failed to expand tilde: got %v", expanded)
	}

	if !filepath.IsAbs(expanded) {
		t.Errorf("ExpandPath() did not return absolute path: got %v", expanded)
	}

	homeDir, _ := os.UserHomeDir()
	expectedPrefix := homeDir
	if !strings.HasPrefix(expanded, expectedPrefix) {
		t.Errorf("ExpandPath() = %v, expected to start with %v", expanded, expectedPrefix)
	}
}

func TestValidate_SSHKeyPathExpansion(t *testing.T) {
	// Create a temporary file to use as SSH key
	tmpDir := t.TempDir()
	tmpKeyFile := filepath.Join(tmpDir, "test_key.pem")

	// Create the file
	f, err := os.Create(tmpKeyFile)
	if err != nil {
		t.Fatalf("Failed to create test key file: %v", err)
	}
	f.Close()

	tests := []struct {
		name        string
		sshKeyPath  string
		shouldExist bool
		wantErr     bool
		description string
	}{
		{
			name:        "Absolute path that exists",
			sshKeyPath:  tmpKeyFile,
			shouldExist: true,
			wantErr:     false,
			description: "Absolute path to existing file should validate",
		},
		{
			name:        "Absolute path that doesn't exist",
			sshKeyPath:  "/nonexistent/path/key.pem",
			shouldExist: false,
			wantErr:     true,
			description: "Non-existent file should fail validation",
		},
		{
			name:        "Empty SSH key path",
			sshKeyPath:  "",
			shouldExist: false,
			wantErr:     true,
			description: "Empty SSH key path should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Provider:          "aws",
				Orchestrator:      "rke2",
				InstanceCount:     1,
				SSHKeyName:        "test-key",
				SSHPrivateKeyPath: tt.sshKeyPath,
			}

			err := cfg.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v (description: %s)", err, tt.wantErr, tt.description)
			}
		})
	}
}

func TestValidate_TildeInSSHPath(t *testing.T) {
	// This test verifies that tilde expansion works in validation
	// We'll use a mock approach by creating a file in home directory

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory, skipping test")
	}

	// Create a temporary test directory in home
	testDir := filepath.Join(homeDir, ".go-k8s-helper-test")
	err = os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a test key file
	testKeyFile := filepath.Join(testDir, "test_key.pem")
	f, err := os.Create(testKeyFile)
	if err != nil {
		t.Fatalf("Failed to create test key file: %v", err)
	}
	f.Close()

	// Test with tilde path
	tildeRelPath := strings.Replace(testKeyFile, homeDir, "~", 1)

	cfg := &Config{
		Provider:          "aws",
		Orchestrator:      "rke2",
		InstanceCount:     1,
		SSHKeyName:        "test-key",
		SSHPrivateKeyPath: tildeRelPath,
	}

	err = cfg.Validate()
	if err != nil {
		t.Errorf("Validate() with tilde path failed: %v (path: %s)", err, tildeRelPath)
	}

	// Verify that the path was expanded in the config
	if strings.HasPrefix(cfg.SSHPrivateKeyPath, "~") {
		t.Errorf("SSHPrivateKeyPath was not expanded: %s", cfg.SSHPrivateKeyPath)
	}

	if cfg.SSHPrivateKeyPath != testKeyFile {
		t.Errorf("SSHPrivateKeyPath = %v, want %v", cfg.SSHPrivateKeyPath, testKeyFile)
	}
}

func TestValidate_RequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Missing provider",
			config: &Config{
				Orchestrator:      "rke2",
				InstanceCount:     1,
				SSHKeyName:        "test-key",
				SSHPrivateKeyPath: "/tmp/test.pem",
			},
			wantErr: true,
			errMsg:  "provider must be specified",
		},
		{
			name: "Missing orchestrator",
			config: &Config{
				Provider:          "aws",
				InstanceCount:     1,
				SSHKeyName:        "test-key",
				SSHPrivateKeyPath: "/tmp/test.pem",
			},
			wantErr: true,
			errMsg:  "orchestrator must be specified",
		},
		{
			name: "Invalid instance count",
			config: &Config{
				Provider:          "aws",
				Orchestrator:      "rke2",
				InstanceCount:     0,
				SSHKeyName:        "test-key",
				SSHPrivateKeyPath: "/tmp/test.pem",
			},
			wantErr: true,
			errMsg:  "instance_count must be at least 1",
		},
		{
			name: "Missing SSH key name",
			config: &Config{
				Provider:          "aws",
				Orchestrator:      "rke2",
				InstanceCount:     1,
				SSHPrivateKeyPath: "/tmp/test.pem",
			},
			wantErr: true,
			errMsg:  "ssh_key_name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, expected to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}
