module my_plugin

go 1.15

replace github.com/jensneuse/graphql-go-tools => github.com/TykTechnologies/graphql-go-tools v1.6.2-0.20220811124354-8d1f142966f8

require (
	github.com/TykTechnologies/tyk v1.9.2-0.20220824110427-3129a8f764b3
	github.com/certifi/gocertifi v0.0.0-20190905060710-a5e0173ced67
	github.com/getkin/kin-openapi v0.89.0
	github.com/gogo/protobuf v1.3.2
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.2
	github.com/vektra/mockery/v2 v2.20.0
	golang.org/x/tools v0.5.0
	google.golang.org/grpc v1.46.2
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0
	google.golang.org/protobuf v1.28.0
)
