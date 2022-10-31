# tyk-spicedb-plugin
Tyk plugin to support SpiceDB

This plugin for Tyk can be used to validate request against a SpiceDB repository. For any API definition which uses the plugin, the parameters passed can be used to verify
Data Access: is the currently logged-in user allowed to perform the operation on that resource.


## Proposal on how to configure the data-access security:
1. Path parameter
```
"/payers/{payerId}": {
    "GET": {
        "payerId": {
            "type": "payer",
            "permission": "view"
    }
}

```

2. Query parameter
```
"/payers?payerId={payerId}: {
    "GET": {
        "payerId": {
            "type": "payer",
            "permission": "view"
    }
```

3. Request body (x-www-urlencoded) `/payers`, payload: `payerId=123&name=John+Doe`
Body parameter:
```
"/programs": {
    "POST": {
        "payerId": {
            "type": "payer",
            "permission": "manage"
    }
```

4. Combined example: `/providers/{providerId}/commonAccounts?payerId={payerId}`
```
"/providers/{providerId}/commonAccounts": {
    "GET": {
        "providerId": {
            "type": "provider",
            "permission": "view"
        },
        "payerId": {
            "type": "payer",
            "permission": "view"
        }
    }
```

# Current setup
```puml
@startuml
title "Login process"
actor User
database DB
User -right-> [UserManagement] : login
User <-right- [UserManagement] : JWT token
[UserManagement] -> [Askari] : getAclEntries
[UserManagement] <- [Askari] : entries
[Askari] <-down- [DB]
@enduml
```

```puml
@startuml
title "Request process"
actor User
User -right-> [CloudFront]
User <-right- [CloudFront]
[CloudFront] -right-> [NLB LoadBalancer]
[CloudFront] <-right- [NLB LoadBalancer]
[NLB LoadBalancer] -right-> [Traefik Daemonset]
[NLB LoadBalancer] <-right- [Traefik Daemonset]
[Traefik Daemonset] -down-> [OTA]
[Traefik Daemonset] <-down- [OTA]
[OTA] -down-> [OPA]
[OTA] <-down- [OPA]
[Traefik Daemonset] -right-> [MicroService]
[Traefik Daemonset] <-right- [MicroService]
@enduml
```
Remarks:
* JWT token contains 'scope' including roles and ACL entries
* OTA implements AuthFwd for Traefik.
* OPA uses 'roles' from JWT token
* data.json contains the RBAC-UI mappings:
* Rule evaluation: intersection of allowed and required roles

```json
{
        "path": "/users/*",
        "method": "PUT",
        "roles": [
          "PAYER_GENERAL_MANAGER",
          "PAYER_USER_MANAGER",
          "PAYER_MEMBERSHIP_MANAGER",
          "PROVIDER_MEDIC",
          "PLATFORM_MEDICAL_MANAGER",
          "SELF_ENROLMENT",
          "DATABASE_CLEANER",
          "DATA_TRANSFER"
        ],
        "enforced": false
}
```

# New setup

```puml
@startuml
actor User
component CloudFront #pink
component LoadBalancer #pink
User -right-> [CloudFront]
User <-right- [CloudFront]
[CloudFront] -right-> [LoadBalancer]
component "EKS Cluster" {
    portIn ingress
    component MicroService #magenta
    component SpiceDB
    component Tyk #aqua {
        component plugin #aqua
        database efs #pink
    }
    ingress -down-> Tyk
    plugin -down-> SpiceDB
    plugin <-down- SpiceDB
    Tyk -right-> MicroService
    Tyk <-right- MicroService
}
[LoadBalancer] -right-> ingress
database s3 #pink
s3 -right-> efs
@enduml
```

# Deployment Diagram

```puml
@startuml
component "EKS Cluster" {
    portIn ingress
    component App1 #magenta
    component App2 #magenta
    component Tyk
    note right of Tyk
      API Gateway
    end note
    component Vault
    note bottom of Vault
      Manages secrets
    end note
    component ArgoCD
    note bottom of ArgoCD
      Manages deployments
    end note
    ingress -down-> Tyk
    Tyk --> App1
    Tyk --> App2
}
@enduml
```


Remarks:
* DataDog logging should be specific when authorization fails: which endpoint, which user, resource and permission required
* Tyk plugin will not support warnings only: when authorization fails, the plugin will return a 403 Forbidden response

POST /payers/{payerId}/contracts?providerId=90
{
"programId": 90,
"consultationPrice": 1000,
"startDate": "2020-01-01",
}

Call to SpiceDB:
3 checks:
1) subject = user:123, resource = payer:300, permission = "manage"
2) subject = user:123, resource = provider:90, permission = "signup"
3) subject = user:123, resource = program:90, permission = "view"

Proposal for configuration the plugin config_data:
1. Tyk Operator and api-definitions:
```yaml
config_data:
  secure-parameters:
  - in: path
    index: 2
    type: payer
    permission: manage
  - in: query
    name: providerId
    type: provider
    permission: signup
  - in: request
    name: programId
    type: program
    permission: view
```

2. Multiple config-files (see above proposal)

Until we have all endpoints configured in the API gateway, we will still allow ingress to the microservice.

# Build and run

## Pre-requisites

- Go lang 1.15
- docker and docker-compose
- SpiceDB CLI 'zed'

## Steps
1. Build the plugin
`docker run -v $(pwd):/plugin-source carepaydev/tyk-plugin-compiler:v4.1.0 middleware/spicedb-plugin.so`. Copy the plugin to the `middleware` directory.
2. Start Tyk and SpiceDB:
`docker compose up -d`
3. Import SpiceDB data:
`zed import authzed-schema.yaml`
4. Use the following api definition:
```json
{
  "name": "Test API",
  "slug": "test-api",
  "api_id": "1",
  "org_id": "1",
  "use_keyless": true,
  "config_data": {
    "secureParameters": [
      { "in": "query", "name": "program", "type": "program", "permission": "view" }
    ]
  },
  "definition": {
    "location": "header",
    "key": "x-api-version"
  },
  "version_data": {
    "not_versioned": true,
    "versions": {
      "Default": {
        "name": "Default",
        "use_extended_paths": true
    }
   }
  },
  "custom_middleware": {
    "post": [
      {
        "name": "SpiceDB",
        "path": "spicedb-plugin.so"
      }
    ],
    "driver": "goplugin"
  },
  "proxy": {
    "listen_path": "/test-api/",
    "target_url": "http://echo.tyk-demo.com:8080/",
    "strip_listen_path": true
  },
  "active": true
}
```
5. Test the API:
We can use the following JWT token: (userId = John)
- `curl -v -s -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJ1c2VySWQiOiJqb2huIn0.mSV62M8YT1_Fq35mJMoemJ4jGKz2LE2C8hb_IqO-9Tc' http://localhost:7391/test-api/get\?program\=mhf`

When we change the 'program' parameter to something else (e.g. no_exist) then the request will be denied.


# Work in progress

Use a configuration file (YAML) to configure the security:

```yaml
    /payers: # /payers?payer=123,456,678,999&payer=999
      GET:
        payer:
          required: true
          type: payer
          permission: view
      POST: # { request: { payer: 123 } }
        payer:
          in: request
          type: payer
          permission: create
    /payers/{payer}:
      GET:
        payer:
          type: payer
          permission: view
    /payers/{payer}/providers:
      GET:
        program:
          in: query
          type: program
          permission: manage
        payer:
          type: payer
          permission: view
    /payers/{payer}/providers/{provider}:
      GET:
        provider:
          type: provider
          permission: view
        payer:
          type: payer
          permission: view
```

Then parse the config-file, and store all rules in a tree:
Each segment of the path is a node in the tree.

GET `/payers/123/providers/456`

1) root: find 'payers' node
2) dynamic '{payer}' node: populate parameter 'payer'
3) static 'providers' node
4) dynamic '{provider}' node: populate parameter 'provider'

When a request comes in, we can use the tree to find the correct rules.

```
payers:
    GET:
    {payer}:
        providers:
        GET:
            program:
            type: program
            permission: view
            payer:
            type: payer
            permission: view
        providers:
        {provider}:
            GET:
            program:
                type: program
                permission: view
            payer:
                type: payer
                permission: view
```

```
type node struct {
    name string
    children map[string]*node
}
type endpoint struct {
    node
    method string
    parameters map[string]secureParameter
}
type parameter struct {
    node
    parameter string
}
type secureParameter struct {
    node
    type string
    permission string
}
```

# Topics to discuss

## Configuration
- CRD with Tyk Operator
- Global config file
- Separate config per api definition

## Algorithm for processing configuration
1) Store in tree

## Algorithm for processing a request
1) Parse the path and extract individual segments
2) For each segment:
    - Check if the segment is a parameter
    - If yes, then add to parameter map (key = parameter name, value = segment value)
    - If no, then continue with the next segment

Deployment:
* each team manages their own set of ACL entries
* Preferrably a git-based workflow
* TEST, ACC and PROD have identical setup
* Workflow for deploying from TEST to ACC to PROD
* Ideally no manual restarts for Tyk when deploying new ACLs

# Alternative deployment option:

* We store OpenAPI specs in 'cp-apis' git repo
* Upon deployment, we deploy by copying files from git to S3
* Cronjob picks up the files from S3 and deploys them to Tyk (using EFS mount)
* Also, the cronjob will trigger the Tyk group reload
* The SpiceDB plugin will find all OAS specs, and open them
* The plugin will parse the OAS specs, and store the rules in a tree

`yq -j r usermanagement-openapi.yaml | curl -X POST 'http://localhost:7391/tyk/apis/oas' -H 'x-tyk-authorization: changeMe' -H 'Content-Type: application/json' --data @-`
Then reload the API definition in Tyk: `curl 'http://localhost:7391/tyk/apis/oas' -H 'x-tyk-authorization: changeMe'`

Handling a request:
* Parse the path and extract individual segments
* For each segment:
   - traverse the tree, find a matching node
   - if no literal match is found, then check if there is a parameter node 
   - For a parameter: add to parameter map (key = parameter name, value = segment value)
* When a request
