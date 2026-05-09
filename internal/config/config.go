package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	CountryName       string
	CaName            string
	OrgName           string
	ServerDomainName  string
	ClientNames       []string
	SharedSAN         string
	CaLifetime        int
	CertLifetime      int
	OutputDir         string
	CaKeyPassword     string
	ClientKeyPassword string
	SkipExisting      bool
}

func Parse() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.CountryName, "country", "CN", "Country name (C)")
	flag.StringVar(&cfg.CaName, "ca-name", "ikev2ca", "CA common name")
	flag.StringVar(&cfg.OrgName, "org", "IKEv2VPN", "Organization name (O)")
	flag.StringVar(&cfg.ServerDomainName, "domain", "", "Server domain name (required)")
	flag.StringVar(&cfg.SharedSAN, "shared-san", "IKEv2Clients", "Shared SAN for iOS/macOS client auth")
	flag.IntVar(&cfg.CaLifetime, "ca-lifetime", 3652, "CA certificate lifetime in days (default ~10 years)")
	flag.IntVar(&cfg.CertLifetime, "cert-lifetime", 18250, "Server/Client cert lifetime in days (default ~50 years)")
	flag.StringVar(&cfg.OutputDir, "output", "./out", "Output directory for certificates")
	flag.StringVar(&cfg.CaKeyPassword, "ca-pass", "", "CA p12 bundle password (required, or set CA_PASS env)")
	flag.StringVar(&cfg.ClientKeyPassword, "client-pass", "", "Client p12 bundle password (required, or set CLIENT_PASS env)")
	flag.BoolVar(&cfg.SkipExisting, "skip-existing", true, "Skip generating client cert if already exists")

	var clientNames string
	flag.StringVar(&clientNames, "clients", "vpnclient", "Client names, space separated")

	flag.Parse()

	if cfg.ServerDomainName == "" {
		fmt.Fprintln(os.Stderr, "Error: -domain is required")
		flag.Usage()
		os.Exit(1)
	}

	if clientNames != "" {
		cfg.ClientNames = strings.Fields(clientNames)
	}

	if cfg.CaKeyPassword == "" {
		cfg.CaKeyPassword = os.Getenv("CA_PASS")
	}
	if cfg.ClientKeyPassword == "" {
		cfg.ClientKeyPassword = os.Getenv("CLIENT_PASS")
	}
	if cfg.CaKeyPassword == "" {
		fmt.Fprintln(os.Stderr, "Error: CA password is required (use -ca-pass or CA_PASS env)")
		os.Exit(1)
	}
	if cfg.ClientKeyPassword == "" {
		fmt.Fprintln(os.Stderr, "Error: Client password is required (use -client-pass or CLIENT_PASS env)")
		os.Exit(1)
	}

	return cfg
}
