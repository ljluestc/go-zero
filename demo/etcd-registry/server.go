package main

import (
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func main() {
	server := rest.MustNewServer(rest.RestConf{
		ServiceConf: service.ServiceConf{
			Name: "demo-api",
		},
		Host: "0.0.0.0",
		Port: 8080,
		Etcd: discov.EtcdConf{
			Hosts: []string{"localhost:2379"}, // etcd endpoints
			Key:   "demo-api",                 // service key in etcd
		},
	})

	// Register API routes
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/v1/hello",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			httpx.OkJson(w, map[string]interface{}{
				"message": "Hello from demo API registered to etcd!",
				"service": "demo-api",
			})
		},
	})

	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/v1/health",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			httpx.OkJson(w, map[string]interface{}{
				"status": "healthy",
				"service": "demo-api",
			})
		},
	})

	fmt.Println("Starting demo REST API server with etcd registration...")
	fmt.Println("API will be registered to etcd with key: demo-api")
	fmt.Println("You can access the API at: http://localhost:8080/api/v1/hello")

	server.Start()
}