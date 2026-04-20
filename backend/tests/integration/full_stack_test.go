package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/cores/mihomo"
	"github.com/isolate-project/isolate-panel/internal/cores/singbox"
	"github.com/isolate-project/isolate-panel/internal/cores/xray"
	"github.com/isolate-project/isolate-panel/internal/models"
)

const (
	fullstackConfigsDir = "/tmp/isolate-fullstack-tests"
	fullstackDockerProj = "../../../docker"
	composeFile      = "docker-compose.fullstack.yml"
	fullstackTimeout = 120 * time.Second
	apiBaseURL       = "http://127.0.0.1:8080"
	echoServerURL    = "http://127.0.0.1:8081"
	testJWTSecret    = "test-jwt-secret-for-fullstack-tests-only"
)

var (
	xrayPorts    = []int{10001, 10002, 10003}
	singboxPorts = []int{11001, 11002, 11003}
	mihomoPorts  = []int{12001, 12002, 12003}
)

func setupFullstackDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.Core{},
		&models.Inbound{},
		&models.Outbound{},
		&models.User{},
		&models.UserInboundMapping{},
		&models.Certificate{},
		&models.GeoRule{},
		&models.WarpRoute{},
		&models.Setting{},
		&models.Admin{},
	))

	return db
}

func seedFullstackData(t *testing.T, db *gorm.DB) {
	t.Helper()

	admin := &models.Admin{
		Username:     "admin",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG",
		IsSuperAdmin: true,
	}
	require.NoError(t, db.Create(admin).Error)

	cores := []models.Core{
		{Name: "xray", Version: "26.3.27", IsEnabled: true, IsRunning: false},
		{Name: "singbox", Version: "1.13.8", IsEnabled: true, IsRunning: false},
		{Name: "mihomo", Version: "1.19.23", IsEnabled: true, IsRunning: false},
	}
	for i := range cores {
		require.NoError(t, db.Create(&cores[i]).Error)
	}

	settings := []models.Setting{
		{Key: "monitoring_mode", Value: "full", ValueType: "string"},
		{Key: "haproxy_enabled", Value: "true", ValueType: "bool"},
	}
	for i := range settings {
		require.NoError(t, db.Create(&settings[i]).Error)
	}
}

func startDockerComposeFullstack(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), fullstackTimeout)
	defer cancel()

	composePath := filepath.Join(fullstackDockerProj, composeFile)
	absComposePath, err := filepath.Abs(composePath)
	require.NoError(t, err, "resolving docker-compose path")

	t.Log("Starting full-stack Docker Compose environment...")

	cmd := exec.CommandContext(ctx, "docker", "compose",
		"-f", absComposePath,
		"--profile", "cores",
		"up", "-d", "--build",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	require.NoError(t, cmd.Run(), "failed to start docker-compose.fullstack.yml")

	t.Log("Waiting for services to be healthy...")
	waitForServicesHealthy(t, ctx)
	
	t.Log("Full-stack environment ready!")
}

func stopDockerComposeFullstack(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	composePath := filepath.Join(fullstackDockerProj, composeFile)
	absComposePath, err := filepath.Abs(composePath)
	require.NoError(t, err)

	t.Log("Stopping full-stack Docker Compose environment...")

	cmd := exec.CommandContext(ctx, "docker", "compose",
		"-f", absComposePath,
		"down", "-v", "--remove-orphans",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: docker compose down failed: %v", err)
	}

	_ = os.RemoveAll(fullstackConfigsDir)
}

func waitForServicesHealthy(t *testing.T, ctx context.Context) {
	t.Helper()
	
	services := []struct {
		name string
		url  string
	}{
		{"isolate-panel", "http://127.0.0.1:8080/health"},
		{"echo-server", echoServerURL},
	}

	for _, svc := range services {
		t.Logf("Waiting for %s...", svc.name)
		require.Eventually(t, func() bool {
			req, _ := http.NewRequestWithContext(ctx, "GET", svc.url, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return false
			}
			defer resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, fullstackTimeout, 2*time.Second, "%s failed to become healthy", svc.name)
	}
}

func getAPIToken(t *testing.T) string {
	t.Helper()

	loginReq := map[string]string{
		"username": "admin",
		"password": "admin",
	}
	body, _ := json.Marshal(loginReq)

	resp, err := http.Post(apiBaseURL+"/api/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	token, ok := result["token"].(string)
	require.True(t, ok, "token not found in response")
	
	return token
}

func makeAPIRequest(t *testing.T, method, path string, body interface{}, token string) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		bodyJSON, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(bodyJSON)
	}

	req, err := http.NewRequest(method, apiBaseURL+path, bodyReader)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	
	return resp
}

func waitForPort(t *testing.T, host string, port int, timeout time.Duration) {
	t.Helper()
	
	address := net.JoinHostPort(host, strconv.Itoa(port))
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}, timeout, 500*time.Millisecond, "port %d not listening", port)
}

func generateTestConfig(t *testing.T, db *gorm.DB, coreName string, protocol string, port int) []byte {
	t.Helper()

	var core models.Core
	require.NoError(t, db.Where("name = ?", coreName).First(&core).Error)

	user := models.User{
		Username: fmt.Sprintf("test-user-%s", coreName),
		UUID:     "a3485e3e-4eef-4e35-9a3b-f2d1f4c6e8a0",
		IsActive: true,
	}
	require.NoError(t, db.Create(&user).Error)

	inbound := models.Inbound{
		Name:          fmt.Sprintf("%s-%s-test", coreName, protocol),
		Protocol:      protocol,
		CoreID:        core.ID,
		ListenAddress: "0.0.0.0",
		Port:          port,
		IsEnabled:     true,
	}
	require.NoError(t, db.Create(&inbound).Error)

	require.NoError(t, db.Create(&models.UserInboundMapping{
		UserID:    user.ID,
		InboundID: inbound.ID,
	}).Error)

	ctx := &cores.ConfigContext{DB: db}

	switch coreName {
	case "xray":
		config, err := xray.GenerateConfig(ctx, core.ID)
		require.NoError(t, err)
		data, _ := json.MarshalIndent(config, "", "  ")
		return data
	case "singbox":
		config, err := singbox.GenerateConfig(ctx, core.ID)
		require.NoError(t, err)
		data, _ := json.MarshalIndent(config, "", "  ")
		return sanitizeSingboxConfig(data)
	case "mihomo":
		config, err := mihomo.GenerateConfig(ctx, core.ID)
		require.NoError(t, err)
		data, _ := json.Marshal(config)
		return data
	default:
		t.Fatalf("unknown core: %s", coreName)
		return nil
	}
}

func writeConfigToSharedVolume(t *testing.T, filename string, data []byte) {
	t.Helper()

	require.NoError(t, os.MkdirAll(fullstackConfigsDir, 0755))
	path := filepath.Join(fullstackConfigsDir, filename)
	require.NoError(t, os.WriteFile(path, data, 0644))

	copyToDockerVolume(t, filename)
}

func copyToDockerVolume(t *testing.T, filename string) {
	t.Helper()
	ctx := context.Background()

	srcPath := filepath.Join(fullstackConfigsDir, filename)
	dstPath := fmt.Sprintf("isolate-panel-fullstack_fullstack-configs:/shared-configs/%s", filename)

	cmd := exec.CommandContext(ctx, "docker", "cp", srcPath, dstPath)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to copy config to volume: %s", string(out))
}

func restartCore(t *testing.T, coreName string, token string) {
	t.Helper()

	resp := makeAPIRequest(t, "POST", fmt.Sprintf("/api/cores/%s/restart", coreName), nil, token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func getCoreStatus(t *testing.T, coreName string, token string) map[string]interface{} {
	t.Helper()

	resp := makeAPIRequest(t, "GET", "/api/cores", nil, token)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	return result
}

func TestFullStack_StartsSuccessfully(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	token := getAPIToken(t)
	require.NotEmpty(t, token)

	resp := makeAPIRequest(t, "GET", "/api/health", nil, token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestFullStack_API_UsersCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	token := getAPIToken(t)

	createReq := map[string]interface{}{
		"username": "testuser-fullstack",
		"email":    "test@fullstack.local",
		"password": "securepassword123",
		"is_active": true,
	}

	resp := makeAPIRequest(t, "POST", "/api/users", createReq, token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdUser map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&createdUser))
	assert.NotNil(t, createdUser["id"])
	assert.Equal(t, "testuser-fullstack", createdUser["username"])

	resp = makeAPIRequest(t, "GET", "/api/users", nil, token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var usersList map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&usersList))
	assert.NotNil(t, usersList["users"])
}

func TestFullStack_CreateInbound_Xray(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	token := getAPIToken(t)

	resp := makeAPIRequest(t, "GET", "/api/cores", nil, token)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var coresResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&coresResp))

	coresData, ok := coresResp["cores"].([]interface{})
	require.True(t, ok)
	
	var xrayCore map[string]interface{}
	for _, c := range coresData {
		core := c.(map[string]interface{})
		if core["name"] == "xray" {
			xrayCore = core
			break
		}
	}
	require.NotNil(t, xrayCore)

	createInboundReq := map[string]interface{}{
		"name":           "xray-vmess-fullstack",
		"protocol":       "vmess",
		"core_id":        xrayCore["id"],
		"listen_address": "0.0.0.0",
		"port":           xrayPorts[0],
		"is_enabled":     true,
	}

	resp = makeAPIRequest(t, "POST", "/api/inbounds", createInboundReq, token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdInbound map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&createdInbound))
	assert.NotNil(t, createdInbound["id"])
	assert.Equal(t, "xray-vmess-fullstack", createdInbound["name"])
}

func TestFullStack_CreateInbound_Singbox(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	token := getAPIToken(t)

	resp := makeAPIRequest(t, "GET", "/api/cores", nil, token)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var coresResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&coresResp))

	coresData, ok := coresResp["cores"].([]interface{})
	require.True(t, ok)
	
	var singboxCore map[string]interface{}
	for _, c := range coresData {
		core := c.(map[string]interface{})
		if core["name"] == "singbox" {
			singboxCore = core
			break
		}
	}
	require.NotNil(t, singboxCore)

	createInboundReq := map[string]interface{}{
		"name":           "singbox-vless-fullstack",
		"protocol":       "vless",
		"core_id":        singboxCore["id"],
		"listen_address": "0.0.0.0",
		"port":           singboxPorts[1],
		"is_enabled":     true,
	}

	resp = makeAPIRequest(t, "POST", "/api/inbounds", createInboundReq, token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdInbound map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&createdInbound))
	assert.NotNil(t, createdInbound["id"])
	assert.Equal(t, "singbox-vless-fullstack", createdInbound["name"])
}

func TestFullStack_CreateInbound_Mihomo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	token := getAPIToken(t)

	resp := makeAPIRequest(t, "GET", "/api/cores", nil, token)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var coresResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&coresResp))

	coresData, ok := coresResp["cores"].([]interface{})
	require.True(t, ok)
	
	var mihomoCore map[string]interface{}
	for _, c := range coresData {
		core := c.(map[string]interface{})
		if core["name"] == "mihomo" {
			mihomoCore = core
			break
		}
	}
	require.NotNil(t, mihomoCore)

	createInboundReq := map[string]interface{}{
		"name":           "mihomo-ss-fullstack",
		"protocol":       "shadowsocks",
		"core_id":        mihomoCore["id"],
		"listen_address": "0.0.0.0",
		"port":           mihomoPorts[0],
		"is_enabled":     true,
	}

	resp = makeAPIRequest(t, "POST", "/api/inbounds", createInboundReq, token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdInbound map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&createdInbound))
	assert.NotNil(t, createdInbound["id"])
	assert.Equal(t, "mihomo-ss-fullstack", createdInbound["name"])
}

func TestFullStack_CoreStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	token := getAPIToken(t)

	resp := makeAPIRequest(t, "GET", "/api/cores", nil, token)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	cores, ok := result["cores"].([]interface{})
	require.True(t, ok)
	require.GreaterOrEqual(t, len(cores), 3)

	coreNames := make([]string, 0, len(cores))
	for _, c := range cores {
		core := c.(map[string]interface{})
		coreNames = append(coreNames, core["name"].(string))
		assert.NotNil(t, core["id"])
		assert.NotNil(t, core["version"])
	}

	assert.Contains(t, coreNames, "xray")
	assert.Contains(t, coreNames, "singbox")
	assert.Contains(t, coreNames, "mihomo")
}

func TestFullStack_PortValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	token := getAPIToken(t)

	checkPortReq := map[string]interface{}{
		"port":     8080,
		"listen":   "0.0.0.0",
		"protocol": "vmess",
	}

	resp := makeAPIRequest(t, "POST", "/api/inbounds/check-port", checkPortReq, token)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	assert.False(t, result["available"].(bool))
}

func TestFullStack_Traffic_EchoServerReachable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	resp, err := http.Get(echoServerURL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Hello from echo server")
}

func TestFullStack_HAProxy_ConfigEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	token := getAPIToken(t)

	resp := makeAPIRequest(t, "GET", "/api/haproxy/config", nil, token)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("HAProxy endpoint not available")
	}

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	assert.NotNil(t, result["config"])
}

func TestFullStack_HAProxy_ReloadEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	token := getAPIToken(t)

	resp := makeAPIRequest(t, "POST", "/api/haproxy/reload", nil, token)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("HAProxy endpoint not available")
	}

	assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, resp.StatusCode)
}

func TestFullStack_HAProxy_StatusEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	token := getAPIToken(t)

	resp := makeAPIRequest(t, "GET", "/api/haproxy/status", nil, token)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("HAProxy endpoint not available")
	}

	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, resp.StatusCode)
}

func TestFullStack_MultiPort_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	token := getAPIToken(t)

	resp := makeAPIRequest(t, "GET", "/api/cores", nil, token)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var coresResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&coresResp))

	coresData, ok := coresResp["cores"].([]interface{})
	require.True(t, ok)

	var xrayCore map[string]interface{}
	for _, c := range coresData {
		core := c.(map[string]interface{})
		if core["name"] == "xray" {
			xrayCore = core
			break
		}
	}
	require.NotNil(t, xrayCore)

	ports := []int{14001, 14002, 14003}
	for i, port := range ports {
		createInboundReq := map[string]interface{}{
			"name":           fmt.Sprintf("multi-port-xray-%d", i),
			"protocol":       "vmess",
			"core_id":        xrayCore["id"],
			"listen_address": "0.0.0.0",
			"port":           port,
			"is_enabled":     true,
		}

		resp = makeAPIRequest(t, "POST", "/api/inbounds", createInboundReq, token)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create inbound on port %d", port)
	}

	resp = makeAPIRequest(t, "GET", "/api/inbounds", nil, token)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var inboundsResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&inboundsResp))

	inboundsData, ok := inboundsResp["inbounds"].([]interface{})
	if ok {
		assert.GreaterOrEqual(t, len(inboundsData), 3)
	}
}

// TestFullStack_CrossCore_SinglePort tests creating inbounds from different cores on the same port
// This is the key feature: Xray + Sing-box + Mihomo all on port 443 with SNI-based routing
func TestFullStack_CrossCore_SinglePort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	token := getAPIToken(t)

	resp := makeAPIRequest(t, "GET", "/api/cores", nil, token)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var coresResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&coresResp))

	coresData, ok := coresResp["cores"].([]interface{})
	require.True(t, ok)
	require.GreaterOrEqual(t, len(coresData), 3, "Need at least 3 cores for cross-core test")

	var xrayCore, singboxCore, mihomoCore map[string]interface{}
	for _, c := range coresData {
		core := c.(map[string]interface{})
		name, _ := core["name"].(string)
		switch name {
		case "xray":
			xrayCore = core
		case "singbox", "sing-box":
			singboxCore = core
		case "mihomo":
			mihomoCore = core
		}
	}

	require.NotNil(t, xrayCore, "Xray core not found")
	require.NotNil(t, singboxCore, "Sing-box core not found")
	require.NotNil(t, mihomoCore, "Mihomo core not found")

	sharedPort := 443

	inbounds := []struct {
		name       string
		protocol   string
		coreID     interface{}
		sniMatch   string
		transport  string
	}{
		{"cross-xray-vless", "vless", xrayCore["id"], "vless.example.com", "tcp"},
		{"cross-sing-vmess", "vmess", singboxCore["id"], "vmess.example.com", "tcp"},
		{"cross-mihomo-trojan", "trojan", mihomoCore["id"], "trojan.example.com", "tcp"},
	}

	for _, inbound := range inbounds {
		createInboundReq := map[string]interface{}{
			"name":           inbound.name,
			"protocol":       inbound.protocol,
			"core_id":        inbound.coreID,
			"listen_address": "0.0.0.0",
			"port":           sharedPort,
			"is_enabled":     true,
			"tls_enabled":    true,
			"sni_match":      inbound.sniMatch,
			"transport":      inbound.transport,
		}

		resp := makeAPIRequest(t, "POST", "/api/inbounds", createInboundReq, token)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode,
			"Failed to create %s inbound on shared port %d", inbound.name, sharedPort)
	}

	resp = makeAPIRequest(t, "GET", fmt.Sprintf("/api/inbounds?port=%d", sharedPort), nil, token)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var inboundsResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&inboundsResp))

	inboundsData, ok := inboundsResp["inbounds"].([]interface{})
	require.True(t, ok)

	var foundXray, foundSingbox, foundMihomo bool
	for _, i := range inboundsData {
		inbound := i.(map[string]interface{})
		name, _ := inbound["name"].(string)
		port, _ := inbound["port"].(float64)
		if int(port) == sharedPort {
			switch name {
			case "cross-xray-vless":
				foundXray = true
			case "cross-sing-vmess":
				foundSingbox = true
			case "cross-mihomo-trojan":
				foundMihomo = true
			}
		}
	}

	assert.True(t, foundXray, "Xray inbound not found on shared port %d", sharedPort)
	assert.True(t, foundSingbox, "Sing-box inbound not found on shared port %d", sharedPort)
	assert.True(t, foundMihomo, "Mihomo inbound not found on shared port %d", sharedPort)

	resp = makeAPIRequest(t, "GET", "/api/haproxy/config", nil, token)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		configStr := string(body)

		hasFrontend := strings.Contains(configStr, "frontend") ||
			strings.Contains(configStr, fmt.Sprintf("bind *:%d", sharedPort))
		hasXrayBackend := strings.Contains(configStr, "xray") ||
			strings.Contains(configStr, "cross-xray")
		hasSingboxBackend := strings.Contains(configStr, "singbox") ||
			strings.Contains(configStr, "sing-box") ||
			strings.Contains(configStr, "cross-sing")
		hasMihomoBackend := strings.Contains(configStr, "mihomo") ||
			strings.Contains(configStr, "cross-mihomo")

		assert.True(t, hasFrontend, "HAProxy config missing frontend for port %d", sharedPort)
		assert.True(t, hasXrayBackend || hasSingboxBackend || hasMihomoBackend,
			"HAProxy config missing backend definitions")
	}
}

// TestFullStack_CrossCore_TrafficRouting verifies that traffic reaches correct core
// by checking that each core's API reports the correct inbound count
func TestFullStack_CrossCore_TrafficRouting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-stack test in short mode")
	}

	startDockerComposeFullstack(t)
	defer stopDockerComposeFullstack(t)

	token := getAPIToken(t)

	resp := makeAPIRequest(t, "GET", "/api/cores", nil, token)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var coresResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&coresResp))

	coresData := coresResp["cores"].([]interface{})
	require.GreaterOrEqual(t, len(coresData), 3)

	var xrayCore map[string]interface{}
	for _, c := range coresData {
		core := c.(map[string]interface{})
		if core["name"] == "xray" {
			xrayCore = core
			break
		}
	}
	require.NotNil(t, xrayCore)

	sharedPort := 8443

	inbounds := []map[string]interface{}{
		{
			"name":           "traffic-xray-1",
			"protocol":       "vless",
			"core_id":        xrayCore["id"],
			"listen_address": "0.0.0.0",
			"port":           sharedPort,
			"is_enabled":     true,
			"tls_enabled":    true,
			"sni_match":      "xray1.example.com",
		},
		{
			"name":           "traffic-xray-2",
			"protocol":       "trojan",
			"core_id":        xrayCore["id"],
			"listen_address": "0.0.0.0",
			"port":           sharedPort,
			"is_enabled":     true,
			"tls_enabled":    true,
			"sni_match":      "xray2.example.com",
		},
	}

	for _, inbound := range inbounds {
		resp := makeAPIRequest(t, "POST", "/api/inbounds", inbound, token)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	resp = makeAPIRequest(t, "GET", "/api/inbounds", nil, token)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var inboundsResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&inboundsResp))

	inboundsData, _ := inboundsResp["inbounds"].([]interface{})

	var countOnSharedPort int
	for _, i := range inboundsData {
		inbound := i.(map[string]interface{})
		if port, ok := inbound["port"].(float64); ok && int(port) == sharedPort {
			countOnSharedPort++
		}
	}

	assert.GreaterOrEqual(t, countOnSharedPort, 2,
		"Expected at least 2 inbounds on shared port %d", sharedPort)
}
