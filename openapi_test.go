package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMakeEntry(t *testing.T) {
	tests := []struct {
		name     string
		segments []string
		isParam  bool
	}{
		{"getUser", []string{"get", "users", "{userId}"}, true},
		{"getUsers", []string{"get", "users"}, false},
		{"deleteProviderContract", []string{"delete", "providers", "{providerId}", "contracts", "{contractId}"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootNode := &OpenApiNode{Paths: make(map[string]*OpenApiNode)}
			n := MakeEntry(rootNode, tt.segments, 0)
			assert.Equal(t, tt.isParam, n.IsPathParam)
			assert.Equal(t, tt.segments[len(tt.segments)-1], n.Name)
			assert.Equal(t, rootNode.Paths[tt.segments[0]].Name, tt.segments[0])
		})
	}
}

func TestReadSpecs(t *testing.T) {
	rootNode, err := ReadSpecs("test-data")
	if err != nil {
		t.Errorf("Failed : %v, :", err)
	}
	getNode, found := rootNode.Paths["get"]
	if !found || getNode.Name != "get" {
		t.Errorf("Expected 'get'")
	}
	node, params, found := FindEntry(rootNode, "get/api/v3/provider/providers/AAR/payment-accounts/123")
	assert.True(t, found)
	assert.NotNil(t, node)
	assert.Contains(t, params, "providerCode", "accountCode")
}

func TestFindEntry(t *testing.T) {
	rootNode, err := ReadSpecs("test-data")
	if err != nil {
		t.Errorf("Failed : %v, :", err)
	}
	tests := []struct {
		name       string
		path       string
		found      bool
		parameters map[string]string
		roles      map[string]bool
	}{
		{
			name:       "getUser",
			path:       "get/api/usermanagement/users/123",
			found:      true,
			roles:      map[string]bool{"PAYER": true, "PROVIDER": true, "PLATFORM_ADMIN": false},
			parameters: map[string]string{"userId": "123"},
		},
		{
			name:       "getUsers",
			path:       "get/api/usermanagement/users",
			found:      true,
			roles:      map[string]bool{"PAYER": false},
			parameters: make(map[string]string),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getNode, parameters, found := FindEntry(rootNode, tt.path)
			if !found {
				if tt.found {
					t.Errorf("Did not find path: %s", tt.path)
				}
			} else {
				if !tt.found {
					t.Errorf("Found path: %s, unexpected", tt.path)
				} else {
					for key, value := range tt.parameters {
						paramValue, found := parameters[key]
						if !found {
							t.Errorf("Expected parameter %s", key)
						} else {
							if paramValue != value {
								t.Errorf("Expected %s, got %s", value, paramValue)
							}
						}
					}
					for role, rolePresent := range tt.roles {
						_, found := getNode.Roles[role]
						if rolePresent && !found {
							t.Errorf("Expected %s role", role)
						} else if !rolePresent && found {
							t.Errorf("Did not expect %s role", role)
						}
					}
				}
			}
		})
	}
}
