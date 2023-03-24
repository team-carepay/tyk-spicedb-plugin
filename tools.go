//go:build tools
// +build tools

package main

//nogo:generate go run github.com/golang/mock/mockgen -destination mocks/client_mock.go -package mocks sigs.k8s.io/controller-runtime/pkg/client Client
//nogo:generate go run github.com/golang/mock/mockgen -destination mocks/manager_mock.go -package mocks sigs.k8s.io/controller-runtime Manager
//nogo:generate go run github.com/vektra/mockery/v2 --srcpkg=sigs.k8s.io/controller-runtime/pkg/client --name=Client
//nogo:generate go run github.com/vektra/mockery/v2 --srcpkg=sigs.k8s.io/controller-runtime/pkg/manager --name=Manager
import (
	_ "github.com/vektra/mockery/v2"
	_ "golang.org/x/tools/cmd/stringer"
)
