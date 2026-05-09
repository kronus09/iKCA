package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/kronus/ikca/internal/certgen"
	"github.com/kronus/ikca/internal/web"
)

func main() {
	mode := flag.String("mode", "web", "Run mode: web or cli")
	listen := flag.String("listen", ":20509", "Web server listen address")
	dataDir := flag.String("data-dir", "./data", "Data directory for persisting certificates")

	domain := flag.String("domain", "", "Server domain name (cli mode)")
	country := flag.String("country", "CN", "Country name")
	org := flag.String("org", "IKEv2VPN", "Organization name")
	caName := flag.String("ca-name", "ikev2ca", "CA common name")
	clientNames := flag.String("clients", "vpnclient", "Client names (space separated)")
	sharedSAN := flag.String("shared-san", "IKEv2Clients", "Shared SAN for iOS/macOS")
	caLifetime := flag.Int("ca-lifetime", 3652, "CA lifetime in days")
	certLifetime := flag.Int("cert-lifetime", 18250, "Cert lifetime in days")
	caPass := flag.String("ca-pass", "", "CA p12 password (or CA_PASS env)")
	clientPass := flag.String("client-pass", "", "Client p12 password (or CLIENT_PASS env)")

	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data dir %s: %v", *dataDir, err)
	}
	web.DataDir = *dataDir

	switch *mode {
	case "cli":
		runCLI(*domain, *country, *org, *caName, *clientNames, *sharedSAN, *caLifetime, *certLifetime, *dataDir, *caPass, *clientPass)
	case "web":
		runWeb(*listen, *dataDir)
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s (use 'web' or 'cli')\n", *mode)
		os.Exit(1)
	}
}

func runCLI(domain, country, org, caName, clientNamesStr, sharedSAN string, caLifetime, certLifetime int, outputDir, caPass, clientPass string) {
	if domain == "" {
		fmt.Fprintln(os.Stderr, "Error: -domain is required in cli mode")
		os.Exit(1)
	}
	if caPass == "" {
		caPass = os.Getenv("CA_PASS")
	}
	if clientPass == "" {
		clientPass = os.Getenv("CLIENT_PASS")
	}
	if caPass == "" {
		fmt.Fprintln(os.Stderr, "Error: CA password required (use -ca-pass or CA_PASS env)")
		os.Exit(1)
	}
	if clientPass == "" {
		fmt.Fprintln(os.Stderr, "Error: Client password required (use -client-pass or CLIENT_PASS env)")
		os.Exit(1)
	}

	clients := strings.Fields(clientNamesStr)

	fmt.Println("========================================")
	fmt.Println("  IKEv2 Certificate Generator (Go)")
	fmt.Println("========================================")
	fmt.Printf("Domain: %s | Clients: %v\n", domain, clients)
	fmt.Printf("Output: %s\n", outputDir)

	result, err := certgen.GenerateAll(country, org, caName, domain, clients, sharedSAN, caLifetime, certLifetime, caPass, clientPass)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := certgen.SaveToDisk(result, outputDir, domain); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nCertificates saved to %s\n", outputDir)
	fmt.Println("1. Upload serverCert_*.pem + serverKey_*.pem to IKEv2 server")
	fmt.Println("2. Import client_*.p12 on client devices")
}

func runWeb(listenAddr, dataDir string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/status", web.HandleStatus)
	mux.HandleFunc("/api/generate", web.HandleGenerate)
	mux.HandleFunc("/api/clear", web.HandleClear)
	mux.HandleFunc("/api/download/ca", web.HandleDownloadCA)
	mux.HandleFunc("/api/download/ca-cert", web.HandleDownloadCACert)
	mux.HandleFunc("/api/download/ca-crt", web.HandleDownloadCACrt)
	mux.HandleFunc("/api/download/server-cert", web.HandleDownloadServerCert)
	mux.HandleFunc("/api/download/server-key", web.HandleDownloadServerKey)
	mux.HandleFunc("/api/download/client", web.HandleDownloadClient)
	mux.HandleFunc("/api/list-data", web.HandleListData)

	staticSub, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticSub)))

	fmt.Printf("IKEv2 Certificate Generator Web UI\n")
	fmt.Printf("Data directory: %s\n", dataDir)
	fmt.Printf("Listening on %s\n", listenAddr)
	url := fmt.Sprintf("http://localhost%s", listenAddr)
	fmt.Printf("\n  ➜  Open: %s\n\n", url)

	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatal(err)
	}
}
