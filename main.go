package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/TykTechnologies/tyk/log"
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

var spiceDbClient *Client
var logger = log.Get()
var rootNode *OpenApiNode

func init() {
	logger.Info("Initializing SpiceDB Plugin")
	endpoint := os.Getenv("AUTHZED_ENDPOINT")
	bearerToken := os.Getenv("AUTHZED_BEARER_TOKEN")
	client, err := NewClient(
		endpoint,
		WithInsecure(),
		WithInsecureBearerToken(bearerToken),
	)
	if err != nil {
		logger.Errorf("unable to initialize client: %s", err)
	}
	spiceDbClient = client
	logger.Infof("SpiceDB Plugin initialized, endpoint: %s, token: %s", endpoint, bearerToken)
	rootNode, err = ReadSpecs("/mnt/tyk-gateway/apps")
	if err != nil {
		logger.Errorf("unable to read specs: %s", err)
	}
}

func ProcessSecureRequest(rw http.ResponseWriter, r *http.Request) {
	logger.
		WithField("url", r.URL.Path).
		WithField("method", r.Method).
		Info("Finding matching path")
	node, pathParams, found := FindEntry(rootNode, strings.ToLower(r.Method)+r.URL.Path)
	if found && len(node.Parameters) > 0 {
		userId, err := extractUserIdFromJwt(r)
		if err != nil {
			logger.
				WithField("url", r.URL.Path).
				WithField("method", r.Method).
				WithField("error", err).
				Error("unable to extract user id from jwt")
			rw.WriteHeader(401) // TODO: allow anonymous endpoints (permitall)
			return
		}
		subject := &SubjectReference{Object: &ObjectReference{ObjectType: "user", ObjectId: userId}}
		for name, param := range node.Parameters {
			var values []string
			if param.In == "path" { // /programs/{programId}/users/{userId}
				values = []string{pathParams[name]}
			} else if param.In == "query" { // /programs?payerId=123
				values = r.URL.Query()[name]
				logger.Info("query value is '", values, "' for ", name)
			} else if param.In == "request" { // /programs   body: payerId=123&name=AAR&description=blabla
				// TODO: suppoort JSON body
				if r.PostForm == nil {
					r.ParseMultipartForm(32 << 20)
				}
				values = r.PostForm[name]
			}
			for _, value := range values {
				resource := &ObjectReference{ObjectType: param.Type, ObjectId: value}
				requestLog := logger.
					WithField("url", r.URL.Path).
					WithField("method", r.Method).
					WithField("parameter", name).
					WithField("in", param.In).
					WithField("user-id", userId).
					WithField("permission", param.Permission).
					WithField("resource-type", param.Type).
					WithField("resource-id", value)
				resp, err2 := spiceDbClient.CheckPermission(r.Context(), &CheckPermissionRequest{
					Resource:   resource,
					Permission: param.Permission,
					Subject:    subject,
				})

				if err2 != nil || resp.Permissionship != CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
					requestLog.Error("Permission denied")
					rw.WriteHeader(403)
					return
				} else {
					requestLog.Info("Permission granted")
				}
			}
		}
	} else {
		logger.
			WithField("url", r.URL.Path).
			WithField("method", r.Method).
			Info("No secure parameters found")
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
