# REST API Etcd Registration Demo

This demo shows how to register REST APIs with etcd and discover them for service calls.

## Prerequisites

1. **etcd**: You need etcd running locally on port 2379. You can start etcd using Docker:

```bash
docker run -d -p 2379:2379 --name etcd quay.io/coreos/etcd:v3.5.9 \
  etcd --advertise-client-urls http://0.0.0.0:2379 \
  --listen-client-urls http://0.0.0.0:2379
```

## Running the Demo

1. **Start the server** (registers the API to etcd):

```bash
cd demo/etcd-registry
go run server.go
```

The server will start on port 8080 and register itself to etcd with the key `demo-api`.

2. **Run the client** (discovers and calls the API):

```bash
cd demo/etcd-registry
go run client.go
```

The client will:
- Discover the API endpoints from etcd
- Call the `/api/v1/hello` endpoint on the discovered service
- Print the response

## Expected Output

**Server output:**
```
Starting demo REST API server with etcd registration...
API will be registered to etcd with key: demo-api
You can access the API at: http://localhost:8080/api/v1/hello
```

**Client output:**
```
Found 1 endpoints for demo-api service:
  1. 127.0.0.1:8080

Calling API at endpoint: 127.0.0.1:8080
Response Status: 200 OK
Response Body: {"message":"Hello from demo API registered to etcd!","service":"demo-api"}
```

## Configuration

### Server Configuration

The server configuration includes etcd settings:

```go
server := rest.MustNewServer(rest.RestConf{
    ServiceConf: rest.ServiceConf{
        Name: "demo-api",
    },
    Host: "0.0.0.0",
    Port: 8080,
    Etcd: discov.EtcdConf{
        Hosts: []string{"localhost:2379"}, // etcd endpoints
        Key:   "demo-api",                 // service key in etcd
    },
})
```

### Optional Etcd Configuration

You can also configure etcd authentication and TLS:

```go
Etcd: discov.EtcdConf{
    Hosts: []string{"localhost:2379"},
    Key:   "demo-api",
    User:  "username",         // optional: etcd username
    Pass:  "password",         // optional: etcd password
    CertFile: "/path/to/cert", // optional: TLS cert file
    CertKeyFile: "/path/to/key", // optional: TLS key file
    CACertFile: "/path/to/ca",   // optional: CA cert file
},
```

## API Endpoints

The demo server exposes two endpoints:

- `GET /api/v1/hello` - Returns a greeting message
- `GET /api/v1/health` - Returns health status

## How It Works

1. **Registration**: The REST server automatically registers its address to etcd when it starts
2. **Discovery**: The client uses a subscriber to watch for service endpoints in etcd
3. **Load Balancing**: Multiple instances can register with the same key for load balancing
4. **Health Checks**: etcd lease mechanism ensures only healthy services are discoverable