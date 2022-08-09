module my_plugin

go 1.15

replace github.com/jensneuse/graphql-go-tools => github.com/TykTechnologies/graphql-go-tools v1.6.2-0.20220426094453-0cc35471c1ca

require (
	github.com/TykTechnologies/tyk v1.9.2-0.20220614105651-6c76e802a298
	github.com/authzed/authzed-go v0.6.0
	github.com/authzed/grpcutil v0.0.0-20220104222419-f813f77722e5
)
