package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	tykctx "github.com/TykTechnologies/tyk/ctx"
	authzedpb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/grpcutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type parameter struct {
	In         string `json:"in"`
	Index      int    `json:"index,omitempty"`
	Name       string `json:"name,omitempty"`
	Type       string `json:"type"`
	Permission string `json:"permission"`
}

var spiceDbClient *authzed.Client

func init() {
	endpoint := os.Getenv("AUTHZED_ENDPOINT")
	bearerToken := os.Getenv("AUTHZED_BEARER_TOKEN")
	client, err := authzed.NewClient(
		endpoint,
		grpcutil.WithSystemCerts(grpcutil.SkipVerifyCA),
		grpcutil.WithBearerToken(bearerToken),
	)
	if err != nil {
		log.Fatalf("unable to initialize client: %s", err)
	}
	spiceDbClient = client
}

func ProcessSecureRequest(rw http.ResponseWriter, r *http.Request) {
	apiDefinition := tykctx.GetDefinition(r)
	secureParams, found := apiDefinition.ConfigData["secureParameters"].([]parameter)
	if found && len(secureParams) > 0 {
		userId, err := extractUserIdFromJwt(r)
		if err != nil {
			rw.WriteHeader(401)
			return
		}
		subject := &authzedpb.SubjectReference{Object: &authzedpb.ObjectReference{ObjectType: "user", ObjectId: userId}}
		pathParts := strings.Split(r.URL.Path, "/")
		for _, param := range secureParams {
			var value string
			if param.In == "path" {
				value = pathParts[param.Index]
			} else if param.In == "query" {
				value = r.Form.Get(param.Name)
			} else if param.In == "request" {
				value = r.PostForm.Get(param.Name)
			}
			resource := &authzedpb.ObjectReference{ObjectType: param.Type, ObjectId: value}
			resp, err2 := spiceDbClient.CheckPermission(r.Context(), &authzedpb.CheckPermissionRequest{
				Resource:   resource,
				Permission: param.Permission,
				Subject:    subject,
			})

			if err2 != nil || resp.Permissionship != authzedpb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
				rw.WriteHeader(403)
			}
		}
	}
}

func extractUserIdFromJwt(request *http.Request) (string, error) {
	authHeader, ok := request.Header["Authorization"]
	if !ok {
		return "", fmt.Errorf("authorization header missing")
	}
	auth := authHeader[0]
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", fmt.Errorf("authorization type not Bearer")
	}
	parts := strings.Split(auth[7:], ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}
	payloadStr, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	d := json.NewDecoder(bytes.NewBuffer(payloadStr))
	d.UseNumber()
	var payload map[string]interface{}
	err = d.Decode(&payload)
	if err != nil {
		return "", err
	}
	// TODO: verify signature
	sub, found := payload["userId"]
	if !found {
		return "", fmt.Errorf("Token is missing sub")
	}
	return sub.(string), nil
}

func main() {}
