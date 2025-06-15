package ssmfile

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

func TestWriteEnvFile(t *testing.T) {
	tests := []struct {
		name    string
		envs    map[string]string
		wantErr bool
		setup   func() string
		verify  func(string) error
	}{
		{
			name: "write multiple environment variables",
			envs: map[string]string{
				"DB_HOST":     "localhost",
				"DB_PORT":     "5432",
				"DB_PASSWORD": "secret",
			},
			setup: func() string {
				f, _ := os.CreateTemp("", "test-env-*")
				return f.Name()
			},
			verify: func(path string) error {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				expected := "DB_HOST=\"localhost\"\nDB_PORT=\"5432\"\nDB_PASSWORD=\"secret\"\n"
				if !containsAll(string(content), expected) {
					return fmt.Errorf("unexpected content: %s", content)
				}
				return nil
			},
		},
		{
			name: "write empty environment map",
			envs: map[string]string{},
			setup: func() string {
				f, _ := os.CreateTemp("", "test-env-*")
				return f.Name()
			},
			verify: func(path string) error {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if len(content) != 0 {
					return fmt.Errorf("expected empty file, got: %s", content)
				}
				return nil
			},
		},
		{
			name: "write to read-only file",
			envs: map[string]string{"TEST": "value"},
			setup: func() string {
				f, _ := os.CreateTemp("", "test-env-*")
				fname := f.Name()
				f.Close()
				os.Chmod(fname, 0444)
				return fname
			},
			wantErr: true,
		},
		{
			name: "write variables with special characters",
			envs: map[string]string{
				"PATH_WITH_SPACES": "value with spaces",
				"QUOTES":           "quoted value",
				"NEWLINES":         "line1\nline2",
			},
			setup: func() string {
				f, _ := os.CreateTemp("", "test-env-*")
				return f.Name()
			},
			verify: func(path string) error {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				expected := "PATH_WITH_SPACES=\"value with spaces\"\nQUOTES=\"quoted value\"\nNEWLINES=\"line1\nline2\""
				if !containsAll(string(content), expected) {
					return fmt.Errorf("unexpected content: %s", content)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			defer os.Remove(path)

			err := writeEnvFile(path, tt.envs)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeEnvFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.verify != nil {
				if err := tt.verify(path); err != nil {
					t.Errorf("verification failed: %v", err)
				}
			}
		})
	}
}

func containsAll(content, expected string) bool {
	lines := strings.SplitSeq(expected, "\n")
	for line := range lines {
		if line != "" && !strings.Contains(content, line) {
			return false
		}
	}
	return true
}
func TestWriteConfigFile(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		value        string
		lastModified time.Time
		wantErr      bool
		setup        func() string
		verify       func(string) error
	}{
		{
			name:         "write config with future timestamp",
			value:        "test config content",
			lastModified: time.Now().Add(24 * time.Hour),
			setup: func() string {
				f, _ := os.CreateTemp("", "test-config-*")
				return f.Name()
			},
			verify: func(path string) error {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if string(content) != "test config content" {
					return fmt.Errorf("unexpected content: %s", content)
				}
				info, err := os.Stat(path)
				if err != nil {
					return err
				}
				if !info.ModTime().After(time.Now()) {
					return fmt.Errorf("expected future modification time")
				}
				return nil
			},
		},
		{
			name:         "write config with past timestamp",
			value:        "historical content",
			lastModified: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			setup: func() string {
				f, _ := os.CreateTemp("", "test-config-*")
				return f.Name()
			},
			verify: func(path string) error {
				info, err := os.Stat(path)
				if err != nil {
					return err
				}
				if !info.ModTime().Equal(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)) {
					return fmt.Errorf("unexpected modification time: %v", info.ModTime())
				}
				return nil
			},
		},
		{
			name:         "write to non-existent directory",
			value:        "test content",
			lastModified: time.Now(),
			setup: func() string {
				return "/nonexistent/directory/config.txt"
			},
			wantErr: true,
		},
		{
			name:         "write empty content",
			value:        "",
			lastModified: time.Now(),
			setup: func() string {
				f, _ := os.CreateTemp("", "test-config-*")
				return f.Name()
			},
			verify: func(path string) error {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if len(content) != 0 {
					return fmt.Errorf("expected empty file, got: %s", content)
				}
				return nil
			},
		},
		{
			name:         "write large content",
			value:        strings.Repeat("large content test ", 1000),
			lastModified: time.Now(),
			setup: func() string {
				f, _ := os.CreateTemp("", "test-config-*")
				return f.Name()
			},
			verify: func(path string) error {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if len(content) != 19000 {
					return fmt.Errorf("unexpected content length: got %d, want 19000", len(content))
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			if !strings.HasPrefix(path, "/nonexistent") {
				defer os.Remove(path)
			}

			err := writeConfigFile(path, tt.value, tt.lastModified)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeConfigFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.verify != nil {
				if err := tt.verify(path); err != nil {
					t.Errorf("verification failed: %v", err)
				}
			}
		})
	}
}
func TestBatcherWriteConfigs(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		setup   func() *Batcher
		mock    func(*mockSSMClient)
		wantErr bool
	}{
		{
			name: "successful write of multiple configs",
			files: map[string]string{
				"/aws/param1": "/tmp/config1",
				"/aws/param2": "/tmp/config2",
			},
			setup: func() *Batcher {
				return &Batcher{
					ssmsvc: &mockSSMClient{},
				}
			},
			mock: func(m *mockSSMClient) {
				m.getParametersFunc = func(ctx context.Context, input *ssm.GetParametersInput) (*ssm.GetParametersOutput, error) {
					return &ssm.GetParametersOutput{
						Parameters: []types.Parameter{
							{
								Name:             aws.String("/aws/param1"),
								Value:            aws.String("value1"),
								Version:          1,
								LastModifiedDate: aws.Time(time.Now()),
							},
							{
								Name:             aws.String("/aws/param2"),
								Value:            aws.String("value2"),
								Version:          2,
								LastModifiedDate: aws.Time(time.Now()),
							},
						},
					}, nil
				}
			},
		},
		{
			name:  "empty parameters map",
			files: map[string]string{},
			setup: func() *Batcher {
				return &Batcher{
					ssmsvc: &mockSSMClient{},
				}
			},
			mock: func(m *mockSSMClient) {
				m.getParametersFunc = func(ctx context.Context, input *ssm.GetParametersInput) (*ssm.GetParametersOutput, error) {
					return &ssm.GetParametersOutput{
						Parameters: []types.Parameter{},
					}, nil
				}
			},
		},
		{
			name: "ssm service error",
			files: map[string]string{
				"/aws/param1": "/tmp/config1",
			},
			setup: func() *Batcher {
				return &Batcher{
					ssmsvc: &mockSSMClient{},
				}
			},
			mock: func(m *mockSSMClient) {
				m.getParametersFunc = func(ctx context.Context, input *ssm.GetParametersInput) (*ssm.GetParametersOutput, error) {
					return nil, fmt.Errorf("SSM service error")
				}
			},
			wantErr: true,
		},
		{
			name: "parameter not found",
			files: map[string]string{
				"/aws/param1": "/tmp/config1",
				"/aws/param2": "/tmp/config2",
			},
			setup: func() *Batcher {
				return &Batcher{
					ssmsvc: &mockSSMClient{},
				}
			},
			mock: func(m *mockSSMClient) {
				m.getParametersFunc = func(ctx context.Context, input *ssm.GetParametersInput) (*ssm.GetParametersOutput, error) {
					return &ssm.GetParametersOutput{
						Parameters: []types.Parameter{
							{
								Name:             aws.String("/aws/param1"),
								Value:            aws.String("value1"),
								Version:          1,
								LastModifiedDate: aws.Time(time.Now()),
							},
						},
						InvalidParameters: []string{"/aws/param2"},
					}, nil
				}
			},
		},
		{
			name: "write permission denied",
			files: map[string]string{
				"/aws/param1": "/root/forbidden/config1",
			},
			setup: func() *Batcher {
				return &Batcher{
					ssmsvc: &mockSSMClient{},
				}
			},
			mock: func(m *mockSSMClient) {
				m.getParametersFunc = func(ctx context.Context, input *ssm.GetParametersInput) (*ssm.GetParametersOutput, error) {
					return &ssm.GetParametersOutput{
						Parameters: []types.Parameter{
							{
								Name:             aws.String("/aws/param1"),
								Value:            aws.String("value1"),
								Version:          1,
								LastModifiedDate: aws.Time(time.Now()),
							},
						},
					}, nil
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batcher := tt.setup()
			if tt.mock != nil {
				tt.mock(batcher.ssmsvc.(*mockSSMClient))
			}

			err := batcher.WriteConfigs(context.Background(), tt.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteConfigs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockSSMClient struct {
	getParametersFunc       func(ctx context.Context, input *ssm.GetParametersInput) (*ssm.GetParametersOutput, error)
	getParametersByPathFunc func(ctx context.Context, input *ssm.GetParametersByPathInput) (*ssm.GetParametersByPathOutput, error)
}

func (m *mockSSMClient) GetParameters(ctx context.Context, input *ssm.GetParametersInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersOutput, error) {
	if m.getParametersFunc != nil {
		return m.getParametersFunc(ctx, input)
	}
	return nil, fmt.Errorf("getParametersFunc not implemented")
}

func (m *mockSSMClient) GetParametersByPath(ctx context.Context, input *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	if m.getParametersByPathFunc != nil {
		return m.getParametersByPathFunc(ctx, input)
	}
	return nil, fmt.Errorf("getParametersByPathFunc not implemented")
}
func TestTrimEnv(t *testing.T) {
	tests := []struct {
		name          string
		parameterName string
		basePath      string
		expected      string
	}{
		{
			name:          "simple path trimming",
			parameterName: "/aws/dev/database/password",
			basePath:      "/aws/dev",
			expected:      "DATABASE_PASSWORD",
		},
		{
			name:          "empty path",
			parameterName: "/test/value",
			basePath:      "",
			expected:      "TEST_VALUE",
		},
		{
			name:          "path equals env",
			parameterName: "/aws/prod",
			basePath:      "/aws/prod",
			expected:      "",
		},
		{
			name:          "multiple leading slashes",
			parameterName: "///test/param",
			basePath:      "/",
			expected:      "TEST_PARAM",
		},
		{
			name:          "path with trailing slash",
			parameterName: "/aws/staging/config",
			basePath:      "/aws/staging",
			expected:      "CONFIG",
		},
		{
			name:          "case conversion",
			parameterName: "/aws/test/mixedCase/param",
			basePath:      "/aws/test",
			expected:      "MIXEDCASE_PARAM",
		},
		{
			name:          "path not in env",
			parameterName: "/different/path/value",
			basePath:      "/aws/test",
			expected:      "DIFFERENT_PATH_VALUE",
		},
		{
			name:          "special characters",
			parameterName: "/aws/test/special-chars_123",
			basePath:      "/aws/test",
			expected:      "SPECIAL-CHARS_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimEnv(tt.parameterName, tt.basePath)
			if result != tt.expected {
				t.Errorf("trimEnv() = %v, want %v", result, tt.expected)
			}
		})
	}
}
