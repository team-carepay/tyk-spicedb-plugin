package main

import (
	"log"
	"testing"
)

func TestMakeEntry(t *testing.T) {
	rootNode := &OpenApiNode{Paths: make(map[string]*OpenApiNode)}
	n := MakeEntry(rootNode, []string{"GET", "users", "{userId}"}, 0)
	if n.Name != "{userId}" {
		log.Fatal("Expected {userId}")
	}
	if n.IsPathParam != true {
		log.Fatal("Should be param")
	}
	getNode, found := rootNode.Paths["GET"]
	if !found || getNode.Name != "GET" {
		log.Fatal("Expected GET")
	}
}

func TestReadSpecs(t *testing.T) {
	rootNode, err := ReadSpecs("test-data")
	if err != nil {
		log.Fatalf("Failed : %v, :", err)
	}
	getNode, found := rootNode.Paths["GET"]
	if !found || getNode.Name != "GET" {
		log.Fatal("Expected GET")
	}
}

func TestFindEntry(t *testing.T) {
	rootNode, err := ReadSpecs("test-data")
	if err != nil {
		log.Fatalf("Failed : %v, :", err)
	}
	getNode, parameters, found := FindEntry(rootNode, "get/api/usermanagement/users/123")
	if !found || getNode.Name != "{userId}" {
		log.Fatal("Expected {userId}")
	} else if parameters["userId"] != "123" {
		log.Fatal("Expected 123")
	}
}
