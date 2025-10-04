package main

import (
	"net/http"
	"testing"
)

func TestParseCustomProviderHeaderRequestPath(t *testing.T) {
	tests := []struct {
		name           string
		requestPath    string
		expectedResult CustomProviderRequestPath
		expectError    bool
	}{
		{
			name:        "valid request path",
			requestPath: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/testing17-rg/providers/Microsoft.CustomProviders/resourceProviders/testing17cp/cyberarkSafes/test-safe-v6-1758822458",
			expectedResult: CustomProviderRequestPath{
				Subscriptions:        "12345678-1234-1234-1234-123456789012",
				ResourceGroups:       "testing17-rg",
				Providers:            "Microsoft.CustomProviders",
				ResourceProviders:    "testing17cp",
				ResourceTypeName:     "cyberarkSafes",
				ResourceInstanceName: "test-safe-v6-1758822458",
			},
			expectError: false,
		},
		{
			name:           "empty request path",
			requestPath:    "",
			expectedResult: CustomProviderRequestPath{},
			expectError:    true,
		},
		{
			name:           "invalid request path - too few segments",
			requestPath:    "/subscriptions",
			expectedResult: CustomProviderRequestPath{},
			expectError:    true,
		},
		{
			name:           "invalid request path - missing segments",
			requestPath:    "/subscriptions/test-sub/resourceGroups",
			expectedResult: CustomProviderRequestPath{},
			expectError:    true,
		},
		{
			name:        "valid request path with trailing slash",
			requestPath: "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.CustomProviders/resourceProviders/test-provider/testAction/testResource/",
			expectedResult: CustomProviderRequestPath{
				Subscriptions:        "test-subscription",
				ResourceGroups:       "test-rg",
				Providers:            "Microsoft.CustomProviders",
				ResourceProviders:    "test-provider",
				ResourceTypeName:     "testAction",
				ResourceInstanceName: "testResource",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			if tt.requestPath != "" {
				req.Header.Set("X-Ms-Customproviders-Requestpath", tt.requestPath)
			}

			result, err := ParseCustomProviderHeaderRequestPath(req)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !tt.expectError {
				if result.Subscriptions != tt.expectedResult.Subscriptions {
					t.Errorf("expected Subscriptions %s, got %s", tt.expectedResult.Subscriptions, result.Subscriptions)
				}
				if result.ResourceGroups != tt.expectedResult.ResourceGroups {
					t.Errorf("expected ResourceGroups %s, got %s", tt.expectedResult.ResourceGroups, result.ResourceGroups)
				}
				if result.Providers != tt.expectedResult.Providers {
					t.Errorf("expected Providers %s, got %s", tt.expectedResult.Providers, result.Providers)
				}
				if result.ResourceProviders != tt.expectedResult.ResourceProviders {
					t.Errorf("expected ResourceProviders %s, got %s", tt.expectedResult.ResourceProviders, result.ResourceProviders)
				}
				if result.ResourceTypeName != tt.expectedResult.ResourceTypeName {
					t.Errorf("expected ResourceTypeName %s, got %s", tt.expectedResult.ResourceTypeName, result.ResourceTypeName)
				}
				if result.ResourceInstanceName != tt.expectedResult.ResourceInstanceName {
					t.Errorf("expected ResourceInstanceName %s, got %s", tt.expectedResult.ResourceInstanceName, result.ResourceInstanceName)
				}
			}
		})
	}
}
func TestCustomProviderRequestPath_String(t *testing.T) {
	tests := []struct {
		name     string
		path     CustomProviderRequestPath
		expected string
	}{
		{
			name: "complete path",
			path: CustomProviderRequestPath{
				Subscriptions:        "12345678-1234-1234-1234-123456789012",
				ResourceGroups:       "testing17-rg",
				Providers:            "Microsoft.CustomProviders",
				ResourceProviders:    "testing17cp",
				ResourceTypeName:     "cyberarkSafes",
				ResourceInstanceName: "test-safe-v6-1758822458",
			},
			expected: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/testing17-rg/providers/Microsoft.CustomProviders/resourceProviders/testing17cp/cyberarkSafes/test-safe-v6-1758822458",
		},
		{
			name:     "empty fields",
			path:     CustomProviderRequestPath{},
			expected: "/subscriptions//resourceGroups//providers//resourceProviders///",
		},
		{
			name: "path with special characters in resource name",
			path: CustomProviderRequestPath{
				Subscriptions:        "test-sub",
				ResourceGroups:       "test-rg",
				Providers:            "Microsoft.CustomProviders",
				ResourceProviders:    "test-provider",
				ResourceTypeName:     "testAction",
				ResourceInstanceName: "resource-with-dashes_and_underscores",
			},
			expected: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.CustomProviders/resourceProviders/test-provider/testAction/resource-with-dashes_and_underscores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.path.ID()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParseCustomProviderHeaderRequestPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		requestPath    string
		expectedResult CustomProviderRequestPath
		expectError    bool
	}{
		{
			name:        "path with extra segments",
			requestPath: "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.CustomProviders/resourceProviders/test-provider/testAction/testResource/extra/segments",
			expectedResult: CustomProviderRequestPath{
				Subscriptions:        "test-sub",
				ResourceGroups:       "test-rg",
				Providers:            "Microsoft.CustomProviders",
				ResourceProviders:    "test-provider",
				ResourceTypeName:     "testAction",
				ResourceInstanceName: "testResource",
			},
			expectError: false,
		},
		{
			name:        "path with empty segments",
			requestPath: "/subscriptions//resourceGroups//providers/Microsoft.CustomProviders/resourceProviders//testAction/testResource",
			expectedResult: CustomProviderRequestPath{
				Subscriptions:        "",
				ResourceGroups:       "",
				Providers:            "Microsoft.CustomProviders",
				ResourceProviders:    "",
				ResourceTypeName:     "testAction",
				ResourceInstanceName: "testResource",
			},
			expectError: false,
		},
		{
			name:           "path with exactly 9 segments (one short)",
			requestPath:    "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.CustomProviders/resourceProviders/test-provider/testAction",
			expectedResult: CustomProviderRequestPath{},
			expectError:    true,
		},
		{
			name:        "path with special characters",
			requestPath: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg_with_underscores/providers/Microsoft.CustomProviders/resourceProviders/test-provider-123/cyberarkSafes/safe.with.dots-and-dashes_123",
			expectedResult: CustomProviderRequestPath{
				Subscriptions:        "12345678-1234-1234-1234-123456789012",
				ResourceGroups:       "test-rg_with_underscores",
				Providers:            "Microsoft.CustomProviders",
				ResourceProviders:    "test-provider-123",
				ResourceTypeName:     "cyberarkSafes",
				ResourceInstanceName: "safe.with.dots-and-dashes_123",
			},
			expectError: false,
		},
		{
			name:        "path with leading and trailing slashes",
			requestPath: "///subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.CustomProviders/resourceProviders/test-provider/testAction/testResource///",
			expectedResult: CustomProviderRequestPath{
				Subscriptions:        "test-sub",
				ResourceGroups:       "test-rg",
				Providers:            "Microsoft.CustomProviders",
				ResourceProviders:    "test-provider",
				ResourceTypeName:     "testAction",
				ResourceInstanceName: "testResource",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			req.Header.Set("X-Ms-Customproviders-Requestpath", tt.requestPath)

			result, err := ParseCustomProviderHeaderRequestPath(req)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !tt.expectError {
				if result.Subscriptions != tt.expectedResult.Subscriptions {
					t.Errorf("expected Subscriptions %s, got %s", tt.expectedResult.Subscriptions, result.Subscriptions)
				}
				if result.ResourceGroups != tt.expectedResult.ResourceGroups {
					t.Errorf("expected ResourceGroups %s, got %s", tt.expectedResult.ResourceGroups, result.ResourceGroups)
				}
				if result.Providers != tt.expectedResult.Providers {
					t.Errorf("expected Providers %s, got %s", tt.expectedResult.Providers, result.Providers)
				}
				if result.ResourceProviders != tt.expectedResult.ResourceProviders {
					t.Errorf("expected ResourceProviders %s, got %s", tt.expectedResult.ResourceProviders, result.ResourceProviders)
				}
				if result.ResourceTypeName != tt.expectedResult.ResourceTypeName {
					t.Errorf("expected ResourceTypeName %s, got %s", tt.expectedResult.ResourceTypeName, result.ResourceTypeName)
				}
				if result.ResourceInstanceName != tt.expectedResult.ResourceInstanceName {
					t.Errorf("expected ResourceInstanceName %s, got %s", tt.expectedResult.ResourceInstanceName, result.ResourceInstanceName)
				}
			}
		})
	}
}
