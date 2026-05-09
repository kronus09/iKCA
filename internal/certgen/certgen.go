package certgen

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

type CertResult struct {
	CA struct {
		CertPEM  string `json:"cert_pem"`
		CertDER  []byte `json:"-"`
		P12      []byte `json:"-"`
		Subject  string `json:"subject"`
		NotAfter string `json:"not_after"`
	} `json:"ca"`
	Server struct {
		CertPEM  string `json:"cert_pem"`
		KeyPEM   string `json:"key_pem"`
		Subject  string `json:"subject"`
		NotAfter string `json:"not_after"`
	} `json:"server"`
	Clients []ClientCertResult `json:"clients"`
}

type ClientCertResult struct {
	Name     string `json:"name"`
	CertDER  []byte `json:"-"`
	P12      []byte `json:"-"`
	Subject  string `json:"subject"`
	NotAfter string `json:"not_after"`
}

func serialNumber() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}

func GenerateCA(keyBits int, country, org, caName string, lifetimeDays int) (*rsa.PrivateKey, *x509.Certificate, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("generate CA key: %w", err)
	}
	sn, err := serialNumber()
	if err != nil {
		return nil, nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          sn,
		Subject:               pkix.Name{Country: []string{country}, Organization: []string{org}, CommonName: caName},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Duration(lifetimeDays) * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create CA cert: %w", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse CA cert: %w", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return key, cert, certPEM, nil
}

func GenerateServerCert(caKey *rsa.PrivateKey, caCert *x509.Certificate, country, org, domain string, lifetimeDays int) (*rsa.PrivateKey, *x509.Certificate, []byte, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("generate server key: %w", err)
	}
	sn, err := serialNumber()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: sn,
		Subject:      pkix.Name{Country: []string{country}, Organization: []string{org}, CommonName: domain},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Duration(lifetimeDays) * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{domain},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("create server cert: %w", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse server cert: %w", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return key, cert, certPEM, keyPEM, nil
}

func GenerateClientCert(caKey *rsa.PrivateKey, caCert *x509.Certificate, country, org, clientName, sharedSAN string, lifetimeDays int) (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generate client key: %w", err)
	}
	sn, err := serialNumber()
	if err != nil {
		return nil, nil, err
	}
	sans := []string{clientName}
	if sharedSAN != "" && !contains(sans, sharedSAN) {
		sans = append(sans, sharedSAN)
	}
	tmpl := &x509.Certificate{
		SerialNumber: sn,
		Subject:      pkix.Name{Country: []string{country}, Organization: []string{org}, CommonName: clientName},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Duration(lifetimeDays) * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		DNSNames:     sans,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create client cert: %w", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("parse client cert: %w", err)
	}
	return key, cert, nil
}

func EncodeP12(key *rsa.PrivateKey, cert *x509.Certificate, caCerts []*x509.Certificate, password string) ([]byte, error) {
	return pkcs12.LegacyRC2.Encode(key, cert, caCerts, password)
}

func GenerateAll(country, org, caName, domain string, clientNames []string, sharedSAN string, caLifetimeDays, certLifetimeDays int, caPass, clientPass string) (*CertResult, error) {
	result := &CertResult{}

	caKey, caCert, caCertPEM, err := GenerateCA(4096, country, org, caName, caLifetimeDays)
	if err != nil {
		return nil, fmt.Errorf("generate CA: %w", err)
	}
	result.CA.CertPEM = string(caCertPEM)
	result.CA.CertDER = caCert.Raw
	result.CA.Subject = caCert.Subject.String()
	result.CA.NotAfter = caCert.NotAfter.Format("2006-01-02")

	caP12, err := EncodeP12(caKey, caCert, []*x509.Certificate{caCert}, caPass)
	if err != nil {
		return nil, fmt.Errorf("encode CA p12: %w", err)
	}
	result.CA.P12 = caP12

	_, serverCert, serverCertPEM, serverKeyPEM, err := GenerateServerCert(caKey, caCert, country, org, domain, certLifetimeDays)
	if err != nil {
		return nil, fmt.Errorf("generate server cert: %w", err)
	}
	result.Server.CertPEM = string(serverCertPEM)
	result.Server.KeyPEM = string(serverKeyPEM)
	result.Server.Subject = serverCert.Subject.String()
	result.Server.NotAfter = serverCert.NotAfter.Format("2006-01-02")

	for _, clientName := range clientNames {
		clientKey, clientCert, err := GenerateClientCert(caKey, caCert, country, org, clientName, sharedSAN, certLifetimeDays)
		if err != nil {
			return nil, fmt.Errorf("generate client cert for %s: %w", clientName, err)
		}
		clientP12, err := EncodeP12(clientKey, clientCert, []*x509.Certificate{caCert}, clientPass)
		if err != nil {
			return nil, fmt.Errorf("encode client p12 for %s: %w", clientName, err)
		}
		cr := ClientCertResult{
			Name:     clientName,
			CertDER:  clientCert.Raw,
			P12:      clientP12,
			Subject:  clientCert.Subject.String(),
			NotAfter: clientCert.NotAfter.Format("2006-01-02"),
		}
		result.Clients = append(result.Clients, cr)
	}

	return result, nil
}

func SaveToDisk(result *CertResult, outputDir, domain string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	if err := os.WriteFile(outputDir+"/caCert.pem", []byte(result.CA.CertPEM), 0644); err != nil {
		return fmt.Errorf("write caCert.pem: %w", err)
	}
	if err := os.WriteFile(outputDir+"/caCert.crt", result.CA.CertDER, 0644); err != nil {
		return fmt.Errorf("write caCert.crt: %w", err)
	}
	if err := os.WriteFile(outputDir+"/ca.p12", result.CA.P12, 0600); err != nil {
		return fmt.Errorf("write ca.p12: %w", err)
	}

	if err := os.WriteFile(outputDir+"/serverCert_"+domain+".pem", []byte(result.Server.CertPEM), 0644); err != nil {
		return fmt.Errorf("write server cert: %w", err)
	}
	if err := os.WriteFile(outputDir+"/serverKey_"+domain+".pem", []byte(result.Server.KeyPEM), 0644); err != nil {
		return fmt.Errorf("write server key: %w", err)
	}

	for _, c := range result.Clients {
		if err := os.WriteFile(outputDir+"/client_"+c.Name+".p12", c.P12, 0600); err != nil {
			return fmt.Errorf("write client %s p12: %w", c.Name, err)
		}
		if err := os.WriteFile(outputDir+"/clientCert_"+c.Name+".crt", c.CertDER, 0644); err != nil {
			return fmt.Errorf("write client %s cert: %w", c.Name, err)
		}
	}

	return nil
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
