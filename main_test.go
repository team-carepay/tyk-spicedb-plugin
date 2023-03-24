package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProcessSecureRequest(t *testing.T) {
	var tests = []struct {
		Name           string
		Method         string
		Path           string
		Claims         string
		err            error
		Status         int
		Permissionship CheckPermissionResponse_Permissionship
		SpiceDbError   error
	}{
		{
			Name:           "valid-roles",
			Method:         http.MethodGet,
			Path:           "api/v3/provider/providers",
			Claims:         `{ "sub": "123", "userId": "123", "exp": 9999999999, "iat": 0, "nbf": 0, "iss": "test", "aud": "test", "jti": "test", "roles": ["PROVIDER"] }`,
			Status:         http.StatusOK,
			Permissionship: CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION,
		},
		{
			Name:           "missing-roles",
			Method:         http.MethodGet,
			Path:           "api/v3/provider/providers",
			Claims:         `{ "sub": "123", "userId": "123", "exp": 9999999999, "iat": 0, "nbf": 0, "iss": "test", "aud": "test", "jti": "test", "roles": ["ACCOUNTHOLDER"] }`,
			Status:         http.StatusForbidden,
			err:            fmt.Errorf("xxx"),
			Permissionship: CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION,
		},
		{
			Name:           "valid-path-parameter",
			Method:         http.MethodGet,
			Path:           "api/v3/provider/providers/AAR/payment-accounts/123",
			Claims:         `{ "sub": "123", "userId": "123", "exp": 9999999999, "iat": 0, "nbf": 0, "iss": "test", "aud": "test", "jti": "test", "roles": ["PAYER"] }`,
			Status:         http.StatusOK,
			Permissionship: CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION,
		},
		{
			Name:           "valid-query-parameter",
			Method:         http.MethodGet,
			Path:           "api/v3/provider/providers?providercodes=123",
			Claims:         `{ "sub": "123", "userId": "123", "exp": 9999999999, "iat": 0, "nbf": 0, "iss": "test", "aud": "test", "jti": "test", "roles": ["PAYER"] }`,
			Status:         http.StatusOK,
			Permissionship: CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION,
		},
		{
			Name:           "noaccess-path-parameter",
			Method:         http.MethodGet,
			Path:           "api/v3/provider/providers/AAR/payment-accounts/123",
			Claims:         "",
			Status:         http.StatusForbidden,
			err:            fmt.Errorf("xxx"),
			Permissionship: CheckPermissionResponse_PERMISSIONSHIP_NO_PERMISSION,
			SpiceDbError:   fmt.Errorf("xxx"),
		},
		{
			Name:           "noaccess-query-parameter",
			Method:         http.MethodGet,
			Path:           "api/v3/provider/providers?providercodes=123",
			Claims:         "",
			Status:         http.StatusForbidden,
			err:            fmt.Errorf("xxx"),
			Permissionship: CheckPermissionResponse_PERMISSIONSHIP_NO_PERMISSION,
			SpiceDbError:   fmt.Errorf("xxx"),
		},
	}

	mockSpiceDBClient := MockPermissionsServiceClient{}
	spiceDbClient = &mockSpiceDBClient
	rootNode, _ = ReadSpecs("test-data")
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx := context.Background()
			recorder := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(ctx, tt.Method, "http://localhost/"+tt.Path, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header["Authorization"] = []string{"Bearer eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9." + base64.RawURLEncoding.EncodeToString([]byte(tt.Claims)) + ".JlX3gXGyClTBFciHhknWrjo7SKqyJ5iBO0n-3S2_I7cIgfaZAeRDJ3SQEbaPxVC7X8aqGCOM-pQOjZPKUJN8DMFrlHTOdqMs0TwQ2PRBmVAxXTSOZOoEhD4ZNCHohYoyfoDhJDP4Qye_FCqu6POJzg0Jcun4d3KW04QTiGxv2PkYqmB7nHxYuJdnqE3704hIS56pc_8q6AW0WIT0W-nIvwzaSbtBU9RgaC7ZpBD2LiNE265UBIFraMDF8IAFw9itZSUCTKg1Q-q27NwwBZNGYStMdIBDor2Bsq5ge51EkWajzZ7ALisVp-bskzUsqUf77ejqX_CBAqkNdH1Zebn93A"}
			mockSpiceDBClient.On("CheckPermission", mock.Anything, mock.Anything).Return(&CheckPermissionResponse{
				Permissionship: tt.Permissionship,
			}, tt.SpiceDbError)

			ProcessSecureRequest(recorder, req)

			if tt.err == nil && recorder.Result().StatusCode != http.StatusOK {
				t.Fatalf("Expected status code 200, got %d: %s", recorder.Result().StatusCode, recorder.Body.String())
			} else if tt.err != nil && recorder.Result().StatusCode == http.StatusOK {
				t.Fatalf("Expected error: %s, got: %d %s", tt.err, recorder.Result().StatusCode, recorder.Body.String())
			}
		})
	}
}

func TestExtractJWT(t *testing.T) {
	var tests = []struct {
		Name            string
		Claims          string
		err             error
		UserId          string
		AllowedRoles    []string
		NotAllowedRoles []string
	}{
		{
			Name:            "valid",
			Claims:          "eyJraWQiOiJUQlB0a0xFUjRrTkMwRTNDNnF1cjkzMVoyMV9LblFQYXQ3T0Z6V0RNRzVRIiwiYWxnIjoiUlM1MTIifQ.eyJzdWIiOiJtLm5pY2hvbHNvbkBjYXJlcGF5LmNvbSIsInJvbGVzIjpbIlBST1ZJREVSX1VTRVJfTUFOQUdFUiIsIlBST1ZJREVSX0FETUlOIl0sInVzZXJJZCI6IjU3Njk4NSIsInR5cGUiOiJVIiwic2NvcGUiOnt9LCJleHAiOjE2NzM2MTkyNDAsImdyYW50cyI6e319.dCjSfFsV9W77ugqJ-FstG33g-Xk0PM3T0JsV0JufcdDwxDd2GY8vLPgMKkOx1n8Bq8wGYd0ABzqUJjnwzXKuBPoKxgvjNuBnIxyWLY5k16_ORI8DiDVmMjmgM4TBpyPhqH9KJhXnw5b7rjQvY4_a5s0ymJ52Xbi3LxDzmseQNCtilVGLhxC1m8EzBsBMypny6f-Bfry3rg4GfdiBCYHLfBId81n_OW60bZwng6IrfBD67VYsGCgGR7Ncc41JYLvpH2BOXzzVgEd_oRhzy-TfvOo38JFZhPLo-4Q4f_L1xgNAz1Ol_6D2h1OE-YmRq5uWumjAst1GwuvkGtLuEV4NUA",
			UserId:          "576985",
			AllowedRoles:    []string{"PROVIDER_USER_MANAGER", "PROVIDER_ADMIN"},
			NotAllowedRoles: []string{"PLATFORM_ADMIN"},
			err:             nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
			r.Header.Add("Authorization", "Bearer "+tt.Claims)
			userId, roles, err := extractJwt(r)
			assert.Equal(t, tt.UserId, userId)
			assert.Equal(t, tt.err, err)
			for _, role := range tt.AllowedRoles {
				assert.Contains(t, roles, role)
			}
			for _, role := range tt.NotAllowedRoles {
				assert.NotContains(t, roles, role)
			}
		})
	}
}
