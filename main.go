package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/TykTechnologies/tyk/log"
	"github.com/sirupsen/logrus"
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

var spiceDbClient PermissionsServiceClient
var logger = log.Get()
var rootNode *OpenApiNode

func init() {
	SecureInit()
}

func SecureInit() {
	logger.Formatter = &logrus.JSONFormatter{}
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
	logger.WithField("rootPath", rootNode).Infof("Parsed OpenAPI specs")

}

func ProcessSecureRequest(rw http.ResponseWriter, r *http.Request) {
	logger.
		WithField("url", r.URL.Path).
		WithField("method", r.Method).
		Info("Finding matching path")
	if spiceDbClient == nil || rootNode == nil {
		SecureInit()
	}
	node, pathParams, found := FindEntry(rootNode, strings.ToLower(r.Method)+r.URL.Path)
	if found && (len(node.Roles) > 0 || len(node.Parameters) > 0) {
		logger.Info("Entry found")
		userId, roles, err := extractJwt(r)
		if err != nil {
			logger.
				WithField("url", r.URL.Path).
				WithField("method", r.Method).
				WithField("error", err).
				Error("unable to extract user id from jwt")
			rw.WriteHeader(http.StatusUnauthorized) // TODO: allow anonymous endpoints (permitall)
			return
		}
		logger.Infof("User-Id %s", userId)
		if len(node.Roles) > 0 {
			var anyRoleFound = false
			for role, _ := range node.Roles {
				_, found := roles[role]
				if found {
					anyRoleFound = true
					break
				}
			}
			if !anyRoleFound {
				missingRoles := make([]string, 0)
				for role, _ := range node.Roles {
					missingRoles = append(missingRoles, role)
				}
				logger.
					WithField("user-id", userId).
					WithField("url", r.URL.Path).
					WithField("method", r.Method).
					WithField("missing-roles", strings.Join(missingRoles, ",")).
					Error("missing role")
				rw.WriteHeader(http.StatusForbidden)
				return
			}
		}
		logger.Info("Roles OK")
		if len(node.Parameters) > 0 {
			subject := &SubjectReference{Object: &ObjectReference{ObjectType: "user", ObjectId: userId}}
			var body map[string]interface{}
			for name, param := range node.Parameters {
				var values []string
				if param.In == "path" { // /programs/{programId}/users/{userId}
					values = []string{pathParams[name]}
				} else if param.In == "query" { // /programs?payerId=123
					values = r.URL.Query()[name]
					logger.Info("query value is '", values, "' for ", name)
				} else if param.In == "request" { // /programs   body: payerId=123&name=AAR&description=blabla
					contentType := r.Header.Get("Content-Type")
					if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") || strings.HasPrefix(contentType, "multipart/form-data") {
						if r.PostForm == nil {
							r.ParseMultipartForm(32 << 20)
						}
						values = r.PostForm[name]
					} else if strings.HasPrefix(contentType, "application/json") {
						if len(body) == 0 {
							decoder := json.NewDecoder(r.Body)
							err := decoder.Decode(&body)
							if err != nil {
								logger.
									WithField("url", r.URL.Path).
									WithField("method", r.Method).
									WithField("error", err).
									Error("unable to decode request body")
								rw.WriteHeader(400)
								return
							}
							body[".parsed"] = true
						}
						value, found := body[name]
						if found {
							values = []string{fmt.Sprintf("%v", value)}
						}
					}
				}
				for _, value := range values {
					resource := &ObjectReference{ObjectType: param.Resource, ObjectId: value}
					requestLog := logger.
						WithField("url", r.URL.Path).
						WithField("method", r.Method).
						WithField("parameter", name).
						WithField("in", param.In).
						WithField("user-id", userId).
						WithField("permission", param.Permission).
						WithField("resource-type", param.Resource).
						WithField("resource-id", value)
					requestLog.Info("Checking permission, calling SpiceDB")
					resp, err2 := spiceDbClient.CheckPermission(r.Context(), &CheckPermissionRequest{
						Resource:   resource,
						Permission: param.Permission,
						Subject:    subject,
					})

					if err2 != nil {
						requestLog.WithError(err2).Error("Error calling SpiceDB")
						rw.WriteHeader(500)
						return
					} else if resp.Permissionship != CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
						requestLog.Warnf("Permission denied")
						rw.WriteHeader(403)
						return
					} else {
						requestLog.Info("Permission granted")
					}
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

func extractJwt(request *http.Request) (string, map[string]void, error) {
	authHeader, ok := request.Header["Authorization"]
	if !ok {
		return "", nil, fmt.Errorf("authorization header missing")
	}
	auth := authHeader[0]
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", nil, fmt.Errorf("authorization type not Bearer")
	}
	parts := strings.Split(auth[7:], ".")
	if len(parts) != 3 {
		return "", nil, fmt.Errorf("invalid token format")
	}
	payloadStr, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", nil, err
	}
	d := json.NewDecoder(bytes.NewBuffer(payloadStr))
	d.UseNumber()
	var payload map[string]interface{}
	err = d.Decode(&payload)
	if err != nil {
		return "", nil, err
	}
	// TODO: verify signature
	sub, found := payload["userId"] // TODO: replace with 'sub'
	if !found {
		return "", nil, fmt.Errorf("Token is missing sub")
	}
	roles, found := payload["roles"]
	roleMap := make(map[string]void)
	for _, v := range roles.([]interface{}) {
		roleMap[v.(string)] = member
	}
	return sub.(string), roleMap, nil
}

func main() {}
