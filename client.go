package main

import "google.golang.org/grpc"

// Client represents an open connection to Authzed.
//
// Clients are backed by a gRPC client and as such are thread-safe.
type Client struct {
	PermissionsServiceClient
}

// NewClient initializes a brand new client for interacting with Authzed.
func NewClient(endpoint string, opts ...grpc.DialOption) (*Client, error) {
	conn, err := grpc.Dial(
		endpoint,
		opts...,
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		NewPermissionsServiceClient(conn),
	}, nil
}
