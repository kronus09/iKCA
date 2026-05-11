package web

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kronus/ikca/internal/certgen"
)

var DataDir = "/app/data"

type GenerateRequest struct {
	Country      string `json:"country"`
	Org          string `json:"org"`
	CaName       string `json:"ca_name"`
	Domain       string `json:"domain"`
	ClientNames  string `json:"client_names"`
	SharedSAN    string `json:"shared_san"`
	CaLifetime   int    `json:"ca_lifetime"`
	CertLifetime int    `json:"cert_lifetime"`
	CaPass       string `json:"ca_pass"`
	ClientPass   string `json:"client_pass"`
}

type SavedConfig struct {
	Country        string        `json:"country"`
	Org            string        `json:"org"`
	CaName         string        `json:"ca_name"`
	Domain         string        `json:"domain"`
	ClientNames    []string      `json:"client_names"`
	SharedSAN      string        `json:"shared_san"`
	CaLifetime     int           `json:"ca_lifetime"`
	CertLifetime   int           `json:"cert_lifetime"`
	CaPass         string        `json:"ca_pass"`
	ClientPass     string        `json:"client_pass"`
	CASubject      string        `json:"ca_subject"`
	CANotAfter     string        `json:"ca_not_after"`
	ServerSubject  string        `json:"server_subject"`
	ServerNotAfter string        `json:"server_not_after"`
	Clients        []SavedClient `json:"clients"`
	GeneratedAt    string        `json:"generated_at"`
}

type SavedClient struct {
	Name     string `json:"name"`
	Subject  string `json:"subject"`
	NotAfter string `json:"not_after"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func configPath() string {
	return filepath.Join(DataDir, "config.json")
}

func saveConfig(req *GenerateRequest, result *certgen.CertResult, clientNames []string) {
	cfg := SavedConfig{
		Country:        req.Country,
		Org:            req.Org,
		CaName:         req.CaName,
		Domain:         req.Domain,
		ClientNames:    clientNames,
		SharedSAN:      req.SharedSAN,
		CaLifetime:     req.CaLifetime,
		CertLifetime:   req.CertLifetime,
		CaPass:         req.CaPass,
		ClientPass:     req.ClientPass,
		CASubject:      result.CA.Subject,
		CANotAfter:     result.CA.NotAfter,
		ServerSubject:  result.Server.Subject,
		ServerNotAfter: result.Server.NotAfter,
		GeneratedAt:    time.Now().Format("2006-01-02 15:04:05"),
	}
	for _, c := range result.Clients {
		cfg.Clients = append(cfg.Clients, SavedClient{Name: c.Name, Subject: c.Subject, NotAfter: c.NotAfter})
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Printf("Warning: marshal config failed: %v", err)
		return
	}
	if err := os.WriteFile(configPath(), data, 0644); err != nil {
		log.Printf("Warning: save config failed: %v", err)
	}
}

func loadConfig() *SavedConfig {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil
	}
	var cfg SavedConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return &cfg
}

func HandleStatus(w http.ResponseWriter, r *http.Request) {
	cfg := loadConfig()
	if cfg == nil {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{"exists": false}})
		return
	}

	if lastResult == nil {
		loadResultFromDisk(cfg)
	}

	type statusResp struct {
		Exists   bool   `json:"exists"`
		*SavedConfig
		ServerCertPEM string `json:"server_cert_pem"`
		ServerKeyPEM  string `json:"server_key_pem"`
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: statusResp{
		Exists:        true,
		SavedConfig:   cfg,
		ServerCertPEM: lastResult.Server.CertPEM,
		ServerKeyPEM:  lastResult.Server.KeyPEM,
	}})
}

func loadResultFromDisk(cfg *SavedConfig) {
	result := &certgen.CertResult{}

	caCertPEM, _ := os.ReadFile(filepath.Join(DataDir, "caCert.pem"))
	caKeyPEM, _ := os.ReadFile(filepath.Join(DataDir, "caKey.pem"))
	result.CA.CertPEM = string(caCertPEM)
	result.CA.KeyPEM = string(caKeyPEM)
	result.CA.Subject = cfg.CASubject
	result.CA.NotAfter = cfg.CANotAfter

	serverCertPEM, _ := os.ReadFile(filepath.Join(DataDir, fmt.Sprintf("serverCert_%s.pem", cfg.Domain)))
	serverKeyPEM, _ := os.ReadFile(filepath.Join(DataDir, fmt.Sprintf("serverKey_%s.pem", cfg.Domain)))
	result.Server.CertPEM = string(serverCertPEM)
	result.Server.KeyPEM = string(serverKeyPEM)
	result.Server.Subject = cfg.ServerSubject
	result.Server.NotAfter = cfg.ServerNotAfter

	if len(caCertPEM) > 0 || len(serverCertPEM) > 0 {
		storeResult(result, cfg.Domain)
	}
}

func HandleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Read body failed"})
		return
	}
	defer r.Body.Close()

	var req GenerateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid JSON: " + err.Error()})
		return
	}

	if req.Domain == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "域名(domain)不能为空"})
		return
	}
	if req.CaPass == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "CA密码不能为空"})
		return
	}
	if req.ClientPass == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "客户端密码不能为空"})
		return
	}

	if req.Country == "" {
		req.Country = "CN"
	}
	if req.Org == "" {
		req.Org = "IKEv2VPN"
	}
	if req.CaName == "" {
		req.CaName = "ikev2ca"
	}
	if req.SharedSAN == "" {
		req.SharedSAN = "IKEv2Clients"
	}
	if req.CaLifetime <= 0 {
		req.CaLifetime = 3652
	}
	if req.CertLifetime <= 0 {
		req.CertLifetime = 18250
	}

	clientNames := strings.Fields(req.ClientNames)
	if len(clientNames) == 0 {
		clientNames = []string{"vpnclient"}
	}

	log.Printf("Generating certs for domain=%s clients=%v", req.Domain, clientNames)

	result, err := certgen.GenerateAll(
		req.Country, req.Org, req.CaName, req.Domain,
		clientNames, req.SharedSAN,
		req.CaLifetime, req.CertLifetime,
		req.CaPass, req.ClientPass,
		DataDir,
	)
	if err != nil {
		log.Printf("Error generating certs: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "证书生成失败: " + err.Error()})
		return
	}

	if err := certgen.SaveToDisk(result, DataDir, req.Domain); err != nil {
		log.Printf("Warning: failed to save certs to data dir: %v", err)
	} else {
		log.Printf("Certs saved to %s", DataDir)
	}

	saveConfig(&req, result, clientNames)

	type responseData struct {
		CA struct {
			CertPEM  string `json:"cert_pem"`
			Subject  string `json:"subject"`
			NotAfter string `json:"not_after"`
		} `json:"ca"`
		Server struct {
			CertPEM  string `json:"cert_pem"`
			KeyPEM   string `json:"key_pem"`
			Subject  string `json:"subject"`
			NotAfter string `json:"not_after"`
		} `json:"server"`
		Clients []struct {
			Name     string `json:"name"`
			CertPEM  string `json:"cert_pem"`
			Subject  string `json:"subject"`
			NotAfter string `json:"not_after"`
		} `json:"clients"`
		DataDir string `json:"data_dir"`
	}

	var resp responseData
	resp.CA.CertPEM = result.CA.CertPEM
	resp.CA.Subject = result.CA.Subject
	resp.CA.NotAfter = result.CA.NotAfter
	resp.Server.CertPEM = result.Server.CertPEM
	resp.Server.KeyPEM = result.Server.KeyPEM
	resp.Server.Subject = result.Server.Subject
	resp.Server.NotAfter = result.Server.NotAfter
	resp.DataDir = DataDir
	for _, c := range result.Clients {
		resp.Clients = append(resp.Clients, struct {
			Name     string `json:"name"`
			CertPEM  string `json:"cert_pem"`
			Subject  string `json:"subject"`
			NotAfter string `json:"not_after"`
		}{Name: c.Name, CertPEM: c.CertPEM, Subject: c.Subject, NotAfter: c.NotAfter})
	}

	storeResult(result, req.Domain)

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: resp})
}

var lastResult *certgen.CertResult
var lastDomain string

func storeResult(result *certgen.CertResult, domain string) {
	lastResult = result
	lastDomain = domain
}

func HandleDownloadCA(w http.ResponseWriter, r *http.Request) {
	if lastResult == nil {
		serveFromDataDir(w, r, "ca.p12")
		return
	}
	w.Header().Set("Content-Type", "application/x-pkcs12")
	w.Header().Set("Content-Disposition", "attachment; filename=ca.p12")
	w.Write(lastResult.CA.P12)
}

func HandleDownloadCACert(w http.ResponseWriter, r *http.Request) {
	if lastResult == nil {
		serveFromDataDir(w, r, "caCert.pem")
		return
	}
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", "attachment; filename=caCert.pem")
	w.Write([]byte(lastResult.CA.CertPEM))
}

func HandleDownloadCACrt(w http.ResponseWriter, r *http.Request) {
	if lastResult != nil {
		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
		w.Header().Set("Content-Disposition", "attachment; filename=caCert.crt")
		w.Write(lastResult.CA.CertDER)
		return
	}
	serveFromDataDir(w, r, "caCert.crt")
}

func HandleDownloadServerCert(w http.ResponseWriter, r *http.Request) {
	if lastResult == nil {
		cfg := loadConfig()
		if cfg != nil {
			serveFromDataDir(w, r, fmt.Sprintf("serverCert_%s.pem", cfg.Domain))
			return
		}
		http.Error(w, "No certificates generated yet", http.StatusNotFound)
		return
	}
	filename := fmt.Sprintf("serverCert_%s.pem", lastDomain)
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Write([]byte(lastResult.Server.CertPEM))
}

func HandleDownloadServerKey(w http.ResponseWriter, r *http.Request) {
	if lastResult == nil {
		cfg := loadConfig()
		if cfg != nil {
			serveFromDataDir(w, r, fmt.Sprintf("serverKey_%s.pem", cfg.Domain))
			return
		}
		http.Error(w, "No certificates generated yet", http.StatusNotFound)
		return
	}
	filename := fmt.Sprintf("serverKey_%s.pem", lastDomain)
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Write([]byte(lastResult.Server.KeyPEM))
}

func HandleDownloadServerCrt(w http.ResponseWriter, r *http.Request) {
	if lastResult != nil {
		filename := fmt.Sprintf("serverCert_%s.crt", lastDomain)
		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		w.Write(lastResult.Server.CertDER)
		return
	}
	cfg := loadConfig()
	if cfg != nil {
		serveFromDataDir(w, r, fmt.Sprintf("serverCert_%s.crt", cfg.Domain))
		return
	}
	http.Error(w, "No certificates generated yet", http.StatusNotFound)
}

func HandleDownloadClient(w http.ResponseWriter, r *http.Request) {
	clientName := r.URL.Query().Get("name")
	if clientName == "" {
		http.Error(w, "Missing client name parameter", http.StatusBadRequest)
		return
	}
	if lastResult != nil {
		for _, c := range lastResult.Clients {
			if c.Name == clientName {
				w.Header().Set("Content-Type", "application/x-pkcs12")
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=client_%s.p12", clientName))
				w.Write(c.P12)
				return
			}
		}
	}
	serveFromDataDir(w, r, fmt.Sprintf("client_%s.p12", clientName))
}

func HandleDownloadClientCert(w http.ResponseWriter, r *http.Request) {
	clientName := r.URL.Query().Get("name")
	if clientName == "" {
		http.Error(w, "Missing client name parameter", http.StatusBadRequest)
		return
	}
	if lastResult != nil {
		for _, c := range lastResult.Clients {
			if c.Name == clientName {
				w.Header().Set("Content-Type", "application/x-x509-ca-cert")
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=clientCert_%s.crt", clientName))
				w.Write(c.CertDER)
				return
			}
		}
	}
	serveFromDataDir(w, r, fmt.Sprintf("clientCert_%s.crt", clientName))
}

func HandleListData(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(DataDir)
	if err != nil {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{"files": []string{}, "dir": DataDir}})
		return
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{"files": files, "dir": DataDir}})
}

func serveFromDataDir(w http.ResponseWriter, r *http.Request, filename string) {
	path := filepath.Join(DataDir, filename)
	http.ServeFile(w, r, path)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func HandleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	entries, err := os.ReadDir(DataDir)
	if err != nil {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Nothing to clear"})
		return
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || e.Name() == ".gitkeep" {
			continue
		}
		if err := os.Remove(filepath.Join(DataDir, e.Name())); err == nil {
			count++
		}
	}
	lastResult = nil
	lastDomain = ""
	log.Printf("Cleared %d files from %s", count, DataDir)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: fmt.Sprintf("已清理 %d 个文件", count)})
}

func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	}
}
