package main

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"io/ioutil"
	"strings"
)

type OasParameter struct {
	Name       string `json:"name"`
	In         string `json:"in"`
	Resource   string `json:"x-type"`
	Permission string `json:"x-permission"`
	JsonPath   string `json:"x-json-path"`
}

type SecureParameter struct {
	Name       string `json:"name"`
	In         string `json:"in"`
	Resource   string `json:"resource"`
	Permission string `json:"permission"`
}

type OasRequestBody struct {
}

type CarepaySecurity struct {
	Roles      []string          `json:"roles"`
	Parameters []SecureParameter `json:"parameters"`
}

type OasOperation struct {
	Parameters []OasParameter `json:"parameters"`
}

type TykListenPath struct {
	Value string `json:"value"`
	Strip bool   `json:"strip"`
}

type TykServer struct {
	ListenPath TykListenPath `json:"listenPath"`
}

type TykConfig struct {
	Server TykServer `json:"server"`
}

type OasSpecification struct {
	Paths     map[string]map[string]OasOperation `json:"paths"`
	TykConfig TykConfig                          `json:"x-tyk-api-gateway"`
}

type OpenApiNode struct {
	Paths       map[string]*OpenApiNode
	Name        string
	IsPathParam bool
	Parameters  map[string]*SecureParameter
	Roles       map[string]void
}

type void struct{}

var member void

func ReadSpecs(path string) (*OpenApiNode, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	} else {
		rootNode := OpenApiNode{Paths: make(map[string]*OpenApiNode)}
		for _, fileInfo := range files {
			if !strings.HasSuffix(fileInfo.Name(), "-oas.json") {
				logger.WithField("file", fileInfo.Name()).Infof("Skipped file")
				continue // only consider openapi json files
			}
			file, err := ioutil.ReadFile(path + "/" + fileInfo.Name())
			if err != nil {
				logger.Errorf("unable to read file: %s", err)
				continue
			}
			doc, err := openapi3.NewLoader().LoadFromData(file)
			if err != nil {
				logger.Errorf("unable to load file: %s", err)
				continue
			}
			tykExtension, found := doc.Extensions["x-tyk-api-gateway"]
			if !found {
				logger.Errorf("unable to find tyk extension")
				continue
			}
			var tykConfig TykConfig
			err = json.Unmarshal(tykExtension.(json.RawMessage), &tykConfig)
			if err != nil {
				logger.Errorf("unable to unmarshal tyk extension: %s", err)
				continue
			}
			for path, pathItem := range doc.Paths {
				for method, operation := range pathItem.Operations() {
					listenPath := strings.TrimSuffix(tykConfig.Server.ListenPath.Value, "/")
					segments := strings.Split(strings.ToLower(method)+listenPath+path, "/") // example: GET /users/{user} => ["get", "users", "{user}"]
					node := MakeEntry(&rootNode, segments, 0)
					xSecurityOperation, found := operation.Extensions["x-security"]
					if found {
						var security CarepaySecurity
						err = json.Unmarshal(xSecurityOperation.(json.RawMessage), &security)
						if err != nil {
							logger.Errorf("unable to unmarshal permission: %s", err)
							continue
						}
						for _, role := range security.Roles {
							node.Roles[role] = member
						}
						for _, parameter := range security.Parameters {
							parameter := parameter // Go hack to force copy
							node.Parameters[parameter.Name] = &parameter
						}
					}
					for _, parameter := range operation.Parameters {
						xSecurityParameter, found := parameter.Value.Extensions["x-security"]
						if found {
							var secureParameter SecureParameter
							err = json.Unmarshal(xSecurityParameter.(json.RawMessage), &secureParameter)
							if err != nil {
								logger.Errorf("unable to unmarshal permission: %s", err)
								continue
							}
							secureParameter.Name = parameter.Value.Name
							secureParameter.In = parameter.Value.In
							node.Parameters[parameter.Value.Name] = &secureParameter
						}
					}
				}
			}
		}
		return &rootNode, nil
	}
}

func MakeEntry(node *OpenApiNode, segments []string, index int) *OpenApiNode {
	if index >= len(segments) {
		return node
	}
	name := segments[index]
	childNode, found := node.Paths[name]
	if !found {
		childNode = &OpenApiNode{Paths: make(map[string]*OpenApiNode), Name: name, Roles: make(map[string]void), Parameters: make(map[string]*SecureParameter)}
		if strings.HasPrefix(name, "{") && strings.HasSuffix(name, "}") {
			childNode.IsPathParam = true
		}
		node.Paths[name] = childNode
	}
	return MakeEntry(childNode, segments, index+1)
}

func FindEntry(rootNode *OpenApiNode, path string) (*OpenApiNode, map[string]string, bool) {
	parameters := make(map[string]string)
	segments := strings.Split(path, "/")
	node, found := findEntry(rootNode, parameters, segments, 0)
	if found {
		return node, parameters, true
	} else {
		return nil, nil, false
	}
}

func findEntry(node *OpenApiNode, parameters map[string]string, segments []string, index int) (*OpenApiNode, bool) {
	if index >= len(segments) {
		return node, true
	}
	name := segments[index]
	childNode, found := node.Paths[name]
	if !found {
		// See if we can find a parameter node
		for _, childNode = range node.Paths {
			if childNode.IsPathParam {
				parameters[childNode.Name[1:len(childNode.Name)-1]] = name
				return findEntry(childNode, parameters, segments, index+1)
			}
		}
		return nil, false
	}
	return findEntry(childNode, parameters, segments, index+1)
}
