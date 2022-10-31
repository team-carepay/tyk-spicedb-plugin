package main

import (
	"encoding/json"
	"io/ioutil"
	"strings"
)

type OasParameter struct {
	Name       string `json:"name"`
	In         string `json:"in"`
	Type       string `json:"x-type"`
	Permission string `json:"x-permission"`
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
	Parameters  map[string]*OasParameter
}

func ReadSpecs(path string) (*OpenApiNode, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	} else {
		rootNode := OpenApiNode{Paths: make(map[string]*OpenApiNode)}
		for _, fileInfo := range files {
			if !strings.HasSuffix(fileInfo.Name(), "-oas.json") {
				continue // only consider openapi json files
			}
			file, err := ioutil.ReadFile(path + "/" + fileInfo.Name())
			if err != nil {
				logger.Errorf("unable to read file: %s", err)
				continue
			}
			var openapi OasSpecification
			err = json.Unmarshal(file, &openapi)
			if err != nil {
				logger.Errorf("unable to parse openapi: %s", err)
				continue
			}
			for path, operations := range openapi.Paths {
				for method, operation := range operations {
					segments := strings.Split(strings.ToLower(method)+openapi.TykConfig.Server.ListenPath.Value+path, "/") // example: GET /users/{user} => ["get", "users", "{user}"]
					node := MakeEntry(&rootNode, segments, 0)
					params := make(map[string]*OasParameter)
					for _, param := range operation.Parameters {
						params[param.Name] = &param
					}
					node.Parameters = params
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
		childNode = &OpenApiNode{Paths: make(map[string]*OpenApiNode), Name: name}
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
