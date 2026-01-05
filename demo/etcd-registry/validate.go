package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func main() {
	fmt.Println("🔍 Validating etcd REST API Registration Implementation")
	fmt.Println(repeat("=", 60))

	// Step 1: Check if etcd is running
	fmt.Println("\n1️⃣ Checking etcd status...")
	if !checkEtcdRunning() {
		fmt.Println("❌ etcd is not running. Please start it first:")
		fmt.Println("   docker run -d -p 2379:2379 --name etcd quay.io/coreos/etcd:v3.5.9 \\")
		fmt.Println("     etcd --advertise-client-urls http://0.0.0.0:2379 \\")
		fmt.Println("     --listen-client-urls http://0.0.0.0:2379")
		os.Exit(1)
	}
	fmt.Println("✅ etcd is running")

	// Step 2: Check if server is running
	fmt.Println("\n2️⃣ Checking if demo server is running...")
	if !checkServerRunning() {
		fmt.Println("❌ Demo server is not running. Please start it first:")
		fmt.Println("   go run server.go")
		os.Exit(1)
	}
	fmt.Println("✅ Demo server is running on port 8080")

	// Step 3: Test direct API call
	fmt.Println("\n3️⃣ Testing direct API call...")
	if !testDirectAPI() {
		fmt.Println("❌ Direct API call failed")
		os.Exit(1)
	}
	fmt.Println("✅ Direct API call successful")

	// Step 4: Check etcd registration
	fmt.Println("\n4️⃣ Checking etcd service registration...")
	if !checkEtcdRegistration() {
		fmt.Println("❌ Service not registered in etcd")
		os.Exit(1)
	}
	fmt.Println("✅ Service properly registered in etcd")

	// Step 5: Test client discovery and call
	fmt.Println("\n5️⃣ Testing client discovery and service call...")
	if !testClientDiscovery() {
		fmt.Println("❌ Client discovery and call failed")
		os.Exit(1)
	}
	fmt.Println("✅ Client discovery and service call successful")

	// Step 6: Test service deregistration on shutdown
	fmt.Println("\n6️⃣ Testing service deregistration...")
	if !testDeregistration() {
		fmt.Println("❌ Service deregistration test failed")
		os.Exit(1)
	}
	fmt.Println("✅ Service deregistration works correctly")

	fmt.Println("\n🎉 All validation tests passed! Implementation is working correctly.")
	fmt.Println("\n📋 Validation Summary:")
	fmt.Println("   ✅ etcd connectivity")
	fmt.Println("   ✅ REST server startup with etcd registration")
	fmt.Println("   ✅ API endpoint functionality")
	fmt.Println("   ✅ etcd service registration")
	fmt.Println("   ✅ Service discovery via client")
	fmt.Println("   ✅ Service deregistration on shutdown")
}

func checkEtcdRunning() bool {
	cmd := exec.Command("docker", "exec", "etcd", "etcdctl", "endpoint", "health")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "healthy")
}

func checkServerRunning() bool {
	cmd := exec.Command("lsof", "-i", ":8080")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), ":8080")
}

func testDirectAPI() bool {
	resp, err := http.Get("http://localhost:8080/api/v1/hello")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func checkEtcdRegistration() bool {
	cmd := exec.Command("docker", "exec", "etcd", "etcdctl", "get", "--prefix", "demo-api")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	outputStr := string(output)
	return strings.Contains(outputStr, "demo-api") && strings.Contains(outputStr, ":8080")
}

func testClientDiscovery() bool {
	cmd := exec.Command("go", "run", "client.go")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Client error: %v\n", err)
		fmt.Printf("Client output: %s\n", string(output))
		return false
	}

	outputStr := string(output)
	return strings.Contains(outputStr, "Found 1 endpoints") &&
		   strings.Contains(outputStr, "Response Status: 200 OK") &&
		   strings.Contains(outputStr, "Hello from demo API registered to etcd")
}

func testDeregistration() bool {
	// Get initial registration count
	initialCount := getEtcdKeyCount()

	// Find and kill the server process
	cmd := exec.Command("pkill", "-f", "go run server.go")
	cmd.Run()

	// Wait for deregistration
	time.Sleep(3 * time.Second)

	// Check if keys were removed
	finalCount := getEtcdKeyCount()

	// Start server again for next tests
	go exec.Command("go", "run", "server.go").Run()
	time.Sleep(2 * time.Second)

	return finalCount < initialCount
}

func getEtcdKeyCount() int {
	cmd := exec.Command("docker", "exec", "etcd", "etcdctl", "get", "--prefix", "demo-api")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "demo-api/") {
			count++
		}
	}
	return count
}
