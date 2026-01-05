package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

func main() {
	fmt.Println("🔍 Simple Validation of etcd REST API Registration")
	fmt.Println(strings.Repeat("=", 55))

	passed := 0
	total := 5

	// 1. Check etcd connectivity
	fmt.Print("\n1. etcd connectivity: ")
	if checkEtcdHealth() {
		fmt.Println("✅ PASS")
		passed++
	} else {
		fmt.Println("❌ FAIL")
	}

	// 2. Check API direct access
	fmt.Print("2. API direct access: ")
	if checkAPIDirect() {
		fmt.Println("✅ PASS")
		passed++
	} else {
		fmt.Println("❌ FAIL")
	}

	// 3. Check etcd registration
	fmt.Print("3. etcd registration: ")
	if checkEtcdRegistration() {
		fmt.Println("✅ PASS")
		passed++
	} else {
		fmt.Println("❌ FAIL")
	}

	// 4. Check client discovery
	fmt.Print("4. client discovery: ")
	if checkClientDiscovery() {
		fmt.Println("✅ PASS")
		passed++
	} else {
		fmt.Println("❌ FAIL")
	}

	// 5. Check code compilation
	fmt.Print("5. code compilation: ")
	if checkCompilation() {
		fmt.Println("✅ PASS")
		passed++
	} else {
		fmt.Println("❌ FAIL")
	}

	fmt.Printf("\n📊 Results: %d/%d tests passed\n", passed, total)

	if passed == total {
		fmt.Println("\n🎉 ALL TESTS PASSED! Implementation is working correctly.")
		fmt.Println("\n✅ What was validated:")
		fmt.Println("   • etcd cluster connectivity")
		fmt.Println("   • REST API server functionality")
		fmt.Println("   • Automatic service registration to etcd")
		fmt.Println("   • Service discovery via etcd")
		fmt.Println("   • Code builds without errors")
	} else {
		fmt.Printf("\n⚠️  %d test(s) failed. Check the output above.\n", total-passed)
	}
}

func checkEtcdHealth() bool {
	cmd := exec.Command("docker", "exec", "etcd", "etcdctl", "endpoint", "health")
	output, err := cmd.CombinedOutput()
	return err == nil && strings.Contains(string(output), "healthy")
}

func checkAPIDirect() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:8080/api/v1/hello")
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

func checkClientDiscovery() bool {
	cmd := exec.Command("go", "run", "client.go")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	outputStr := string(output)
	return strings.Contains(outputStr, "Found") &&
		   strings.Contains(outputStr, "Response Status: 200 OK") &&
		   strings.Contains(outputStr, "Hello from demo API")
}

func checkCompilation() bool {
	// Test that the main go-zero rest package builds correctly
	cmd := exec.Command("go", "build", "-o", "/tmp/test_build", "github.com/zeromicro/go-zero/rest")
	err := cmd.Run()
	if err != nil {
		return false
	}
	// Clean up
	exec.Command("rm", "-f", "/tmp/test_build").Run()
	return true
}
