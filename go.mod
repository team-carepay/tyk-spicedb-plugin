module my_plugin

go 1.15

replace github.com/jensneuse/graphql-go-tools => github.com/TykTechnologies/graphql-go-tools v1.6.2-0.20220811124354-8d1f142966f8

require (
	github.com/TykTechnologies/tyk v1.9.2-0.20220824110427-3129a8f764b3
	github.com/certifi/gocertifi v0.0.0-20190905060710-a5e0173ced67
	github.com/mitchellh/mapstructure v1.4.1
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.27.1
)
