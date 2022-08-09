# tyk-spicedb-plugin
Tyk plugin to support SpiceDB

This plugin for Tyk can be used to validate request against a SpiceDB repository. For any API definition which uses the plugin, the parameters passed can be used to verify
Data Access: is the currently logged-in user allowed to perform the operation on that resource.


Examples:
1. Path parameter `/payers/{payerId}`
```
config_data:
  secureParameters:
    - in: path
      index: 2
      type: payer
      permission: view
```

2. Query parameter `/payers?payerId={payerId}`
```
proxy:
  listen_path: /payers
  target_url: http://httpbin.org/
  strip_listen_path: true
custom_middleware:
  driver: golang
  auth_check:
    path: plugins/my-post-example.so
config_data:
  secureParameters:
    - in: query
      name: payerId
      type: payer
      permission: view
```

3. Request body (x-www-urlencoded) `/payers`, payload: `payerId=123&name=John+Doe`
Body parameter:
```
config_data:
  secureParameters:
    - in: request
      name: payerId
      type: payer
      permission: view
```

4. Request body (JSON): `/payers`, JSON payload: `{ "payerId": "123", "name": "John Doe" }`
Body parameter:
```
config_data:
  secureParameters:
    - in: request
      name: payerId
      type: payer
      permission: view
```

5. Combined example: `/providers/{providerId}/commonAccounts?payerId={payerId}`
```
config_data:
  secureParameters:
    - in: query
      name: payerId
      type: payer
      permission: view_payer
    - in: path
      index: 2
      type: provider
      permission: start_treatment
```
