# api-ratelimit

Quick Start for Enroute Universal API Gateway Advanced Rate Limiting 

Article here: https://getenroute.io/cookbook/getting-started-advanced-rate-limiting/

# Usage:

```
go run api-rate-limit.go --op=create --dbg=true
go run api-rate-limit.go --op=show --dbg=true
go run api-rate-limit.go --op=delete --dbg=true

```
# Generate Traffic:

```
curl -vvv -H "x-app-key: hdr-app-saaras" http://localhost:8080/?api-key=query-param-saaras
```
