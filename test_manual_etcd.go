package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func main() {
	fmt.Println("🔬 Manual Testing: etcd REST API Registration")
	fmt.Println(repeat("=", 50))

	// Step 1: Start etcd
	fmt.Println("\n📋 Step 1: Starting etcd...")
	if err := startEtcd(); err != nil {
		fmt.Printf("❌ Failed to start etcd: %v\n", err)
		fmt.Println("Please run manually: docker run -d -p 2379:2379 --name etcd quay.io/coreos/etcd:v3.5.9 etcd --advertise-client-urls http://0.0.0.0:2379 --listen-client-urls http://0.0.0.0:2379")
		os.Exit(1)
	}
	fmt.Println("✅ etcd started successfully")

	// Step 2: Start test server
	fmt.Println("\n📋 Step 2: Starting test REST server with etcd registration...")
	go startTestServer()
	time.Sleep(3 * time.Second)

	// Step 3: Verify registration
	fmt.Println("\n📋 Step 3: Verifying etcd registration...")
	if !verifyRegistration() {
		fmt.Println("❌ Service registration failed")
		os.Exit(1)
	}
	fmt.Println("✅ Service registered in etcd")

	// Step 4: Test API access
	fmt.Println("\n📋 Step 4: Testing API access...")
	if !testAPI() {
		fmt.Println("❌ API access failed")
		os.Exit(1)
	}
	fmt.Println("✅ API responding correctly")

	// Step 5: Test service discovery
	fmt.Println("\n📋 Step 5: Testing service discovery...")
	if !testDiscovery() {
		fmt.Println("❌ Service discovery failed")
		os.Exit(1)
	}
	fmt.Println("✅ Service discovery working")

	fmt.Println("\n🎉 All manual tests passed!")
	fmt.Println("\n📊 Test Results:")
	fmt.Println("   ✅ etcd connectivity")
	fmt.Println("   ✅ REST server startup with etcd registration")
	fmt.Println("   ✅ etcd service registration")
	fmt.Println("   ✅ API endpoint functionality")
	fmt.Println("   ✅ Service discovery via etcd")

	// Cleanup
	fmt.Println("\n🧹 Cleaning up...")
	stopEtcd()
	fmt.Println("✅ Cleanup complete")
}

func startEtcd() error {
	// Check if already running
	if checkEtcdRunning() {
		return nil
	}

	cmd := exec.Command("docker", "run", "-d", "--name", "etcd", "-p", "2379:2379",
		"quay.io/coreos/etcd:v3.5.9",
		"etcd", "--advertise-client-urls", "http://0.0.0.0:2379",
		"--listen-client-urls", "http://0.0.0.0:2379")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start etcd: %v, output: %s", err, string(output))
	}

	// Wait for etcd to be ready
	time.Sleep(3 * time.Second)

	if !checkEtcdRunning() {
		return fmt.Errorf("etcd failed to start properly")
	}

	return nil
}

func checkEtcdRunning() bool {
	cmd := exec.Command("docker", "exec", "etcd", "etcdctl", "endpoint", "health")
	err := cmd.Run()
	return err == nil
}

func startTestServer() {
	server := rest.MustNewServer(rest.RestConf{
		ServiceConf: service.ServiceConf{Name: "test-api"},
		Host:        "0.0.0.0",
		Port:        8080,
		Etcd: discov.EtcdConf{
			Hosts: []string{"localhost:2379"},
			Key:   "test-api",
		},
	})

	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/health",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			httpx.OkJson(w, map[string]string{"status": "ok", "service": "test-api"})
		},
	})

	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/v1/test",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			httpx.OkJson(w, map[string]interface{}{
				"message": "Hello from etcd-registered REST API!",
				"timestamp": time.Now().Unix(),
			})
		},
	})

	fmt.Println("🚀 Test server starting on :8080 with etcd registration...")
	server.Start()
}

func verifyRegistration() bool {
	cmd := exec.Command("docker", "exec", "etcd", "etcdctl", "get", "--prefix", "test-api")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("etcdctl error: %v\n", err)
		return false
	}

	outputStr := string(output)
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")

	if len(lines) < 2 {
		fmt.Printf("Expected at least 2 lines, got: %d\n", len(lines))
		return false
	}

	// Check if we have key and value
	hasKey := false
	hasValue := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "test-api/") {
			hasKey = true
		}
		if strings.Contains(line, ":8080") {
			hasValue = true
		}
	}

	fmt.Printf("Registration check - Key: %v, Value: %v\n", hasKey, hasValue)
	return hasKey && hasValue
}

func testAPI() bool {
	client := &http.Client{Timeout: 5 * time.Second}

	// Test health endpoint
	resp, err := client.Get("http://localhost:8080/health")
	if err != nil {
		fmt.Printf("Health endpoint error: %v\n", err)
		return false
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Printf("Health endpoint status: %d\n", resp.StatusCode)
		return false
	}

	// Test API endpoint
	resp, err = client.Get("http://localhost:8080/api/v1/test")
	if err != nil {
		fmt.Printf("API endpoint error: %v\n", err)
		return false
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Printf("API endpoint status: %d\n", resp.StatusCode)
		return false
	}

	return true
}

func testDiscovery() bool {
	subscriber, err := discov.NewSubscriber(
		[]string{"localhost:2379"},
		"test-api",
	)
	if err != nil {
		fmt.Printf("Subscriber creation error: %v\n", err)
		return false
	}

	// Wait for discovery
	time.Sleep(3 * time.Second)

	endpoints := subscriber.Values()
	if len(endpoints) == 0 {
		fmt.Println("No endpoints discovered")
		return false
	}

	fmt.Printf("Discovered endpoints: %v\n", endpoints)

	// Verify endpoint format
	for _, endpoint := range endpoints {
		if !strings.Contains(endpoint, ":8080") {
			fmt.Printf("Invalid endpoint format: %s\n", endpoint)
			return false
		}
	}

	return true
}

func stopEtcd() {
	exec.Command("docker", "stop", "etcd").Run()
	exec.Command("docker", "rm", "etcd").Run()
}
