package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"ikca-desktop/certgen"
)

type App struct {
	ctx     context.Context
	dataDir string
}

func NewApp() *App {
	exePath, _ := os.Executable()
	dataDir := filepath.Join(filepath.Dir(exePath), "data")
	return &App{dataDir: dataDir}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

func (a *App) OpenExternalLink(url string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("cmd", "/c", "start", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}

func (a *App) GetDataDir() string {
	return a.dataDir
}

type GenerateParams struct {
	Country      string   `json:"country"`
	Org          string   `json:"org"`
	CaName       string   `json:"ca_name"`
	Domain       string   `json:"domain"`
	ClientNames  []string `json:"client_names"`
	SharedSAN    string   `json:"shared_san"`
	CaLifetime   int      `json:"ca_lifetime"`
	CertLifetime int      `json:"cert_lifetime"`
	CaPass       string   `json:"ca_pass"`
	ClientPass   string   `json:"client_pass"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type SavedConfig struct {
	Country        string   `json:"country"`
	Org            string   `json:"org"`
	CaName         string   `json:"ca_name"`
	Domain         string   `json:"domain"`
	ClientNames    []string `json:"client_names"`
	SharedSAN      string   `json:"shared_san"`
	CaLifetime     int      `json:"ca_lifetime"`
	CertLifetime   int      `json:"cert_lifetime"`
	CaPass         string   `json:"ca_pass"`
	ClientPass     string   `json:"client_pass"`
	CASubject      string   `json:"ca_subject"`
	CANotAfter     string   `json:"ca_not_after"`
	ServerSubject  string   `json:"server_subject"`
	ServerNotAfter string   `json:"server_not_after"`
	Clients        []struct {
		Name     string `json:"name"`
		Subject  string `json:"subject"`
		NotAfter string `json:"not_after"`
	} `json:"clients"`
	GeneratedAt string `json:"generated_at"`
}

func (a *App) saveConfig(params *GenerateParams, result *certgen.CertResult) {
	cfg := SavedConfig{
		Country:        params.Country,
		Org:            params.Org,
		CaName:         params.CaName,
		Domain:         params.Domain,
		ClientNames:    params.ClientNames,
		SharedSAN:      params.SharedSAN,
		CaLifetime:     params.CaLifetime,
		CertLifetime:   params.CertLifetime,
		CaPass:         params.CaPass,
		ClientPass:     params.ClientPass,
		CASubject:      result.CA.Subject,
		CANotAfter:     result.CA.NotAfter,
		ServerSubject:  result.Server.Subject,
		ServerNotAfter: result.Server.NotAfter,
		GeneratedAt:    "just now",
	}
	for _, c := range result.Clients {
		cfg.Clients = append(cfg.Clients, struct {
			Name     string `json:"name"`
			Subject  string `json:"subject"`
			NotAfter string `json:"not_after"`
		}{Name: c.Name, Subject: c.Subject, NotAfter: c.NotAfter})
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(a.dataDir, "config.json"), data, 0644)
}

func (a *App) LoadConfig() *SavedConfig {
	data, err := os.ReadFile(filepath.Join(a.dataDir, "config.json"))
	if err != nil {
		return nil
	}
	var cfg SavedConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return &cfg
}

func (a *App) Status() APIResponse {
	cfg := a.LoadConfig()
	if cfg == nil {
		return APIResponse{Success: true, Data: map[string]interface{}{"exists": false}}
	}

	serverCertPEM, _ := os.ReadFile(filepath.Join(a.dataDir, "serverCert_"+cfg.Domain+".pem"))
	serverKeyPEM, _ := os.ReadFile(filepath.Join(a.dataDir, "serverKey_"+cfg.Domain+".pem"))

	type statusData struct {
		Exists         bool   `json:"exists"`
		ServerCertPEM  string `json:"server_cert_pem"`
		ServerKeyPEM   string `json:"server_key_pem"`
		*SavedConfig
	}

	return APIResponse{Success: true, Data: statusData{
		Exists:        true,
		ServerCertPEM: string(serverCertPEM),
		ServerKeyPEM:  string(serverKeyPEM),
		SavedConfig:   cfg,
	}}
}

func (a *App) Generate(params GenerateParams) APIResponse {
	if params.Domain == "" {
		return APIResponse{Success: false, Message: "域名不能为空"}
	}
	if params.CaPass == "" {
		return APIResponse{Success: false, Message: "CA密码不能为空"}
	}
	if params.ClientPass == "" {
		return APIResponse{Success: false, Message: "客户端密码不能为空"}
	}
	if params.Country == "" {
		params.Country = "CN"
	}
	if params.Org == "" {
		params.Org = "IKEv2VPN"
	}
	if params.CaName == "" {
		params.CaName = "ikev2ca"
	}
	if params.SharedSAN == "" {
		params.SharedSAN = "IKEv2Clients"
	}
	if params.CaLifetime <= 0 {
		params.CaLifetime = 3652
	}
	if params.CertLifetime <= 0 {
		params.CertLifetime = 18250
	}
	if len(params.ClientNames) == 0 {
		params.ClientNames = []string{"vpnclient"}
	}

	if err := os.MkdirAll(a.dataDir, 0755); err != nil {
		return APIResponse{Success: false, Message: "创建数据目录失败: " + err.Error()}
	}

	result, err := certgen.GenerateAll(
		params.Country, params.Org, params.CaName, params.Domain,
		params.ClientNames, params.SharedSAN,
		params.CaLifetime, params.CertLifetime,
		params.CaPass, params.ClientPass,
		a.dataDir,
	)
	if err != nil {
		return APIResponse{Success: false, Message: "证书生成失败: " + err.Error()}
	}

	if err := certgen.SaveToDisk(result, a.dataDir, params.Domain); err != nil {
		return APIResponse{Success: false, Message: "证书保存失败: " + err.Error()}
	}

	a.saveConfig(&params, result)

	type clientInfo struct {
		Name     string `json:"name"`
		Subject  string `json:"subject"`
		NotAfter string `json:"not_after"`
	}
	type respData struct {
		CASubject      string       `json:"ca_subject"`
		CANotAfter     string       `json:"ca_not_after"`
		ServerSubject  string       `json:"server_subject"`
		ServerNotAfter string       `json:"server_not_after"`
		ServerCertPEM  string       `json:"server_cert_pem"`
		ServerKeyPEM   string       `json:"server_key_pem"`
		Clients        []clientInfo `json:"clients"`
		DataDir        string       `json:"data_dir"`
	}

	var clients []clientInfo
	for _, c := range result.Clients {
		clients = append(clients, clientInfo{Name: c.Name, Subject: c.Subject, NotAfter: c.NotAfter})
	}

	return APIResponse{Success: true, Data: respData{
		CASubject:      result.CA.Subject,
		CANotAfter:     result.CA.NotAfter,
		ServerSubject:  result.Server.Subject,
		ServerNotAfter: result.Server.NotAfter,
		ServerCertPEM:  result.Server.CertPEM,
		ServerKeyPEM:   result.Server.KeyPEM,
		Clients:        clients,
		DataDir:        a.dataDir,
	}}
}

func (a *App) Clear() APIResponse {
	entries, err := os.ReadDir(a.dataDir)
	if err != nil {
		return APIResponse{Success: true, Message: "Nothing to clear"}
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || e.Name() == ".gitkeep" {
			continue
		}
		if err := os.Remove(filepath.Join(a.dataDir, e.Name())); err == nil {
			count++
		}
	}
	return APIResponse{Success: true, Message: "ok"}
}

func (a *App) OpenDataDir() {
	os.MkdirAll(a.dataDir, 0755)
	switch runtime.GOOS {
	case "windows":
		exec.Command("explorer", a.dataDir).Start()
	case "darwin":
		exec.Command("open", a.dataDir).Start()
	default:
		exec.Command("xdg-open", a.dataDir).Start()
	}
}

func (a *App) GetVersion() string {
	return "0.1.0"
}
