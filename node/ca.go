package node

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// CA holds a loaded Certificate Authority keypair.
type CA struct {
	CACert *x509.Certificate
	CAKey  *ecdsa.PrivateKey
}

// LoadOrCreateCA loads a CA from certDir/ca.crt + certDir/ca.key, or generates a
// new self-signed root CA if those files don't exist.
// The private key is stored in plain PEM (protect with filesystem permissions);
// pass non-nil kek to AES-GCM encrypt it on disk (stored as ca.key.enc).
func LoadOrCreateCA(certDir string, kek []byte) (*CA, error) {
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return nil, fmt.Errorf("create cert dir: %w", err)
	}

	certPath := filepath.Join(certDir, "ca.crt")
	keyPath := filepath.Join(certDir, "ca.key")

	if _, err := os.Stat(certPath); err == nil {
		// Load existing CA.
		certPEM, err := os.ReadFile(certPath)
		if err != nil {
			return nil, fmt.Errorf("read ca cert: %w", err)
		}
		keyPEM, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read ca key: %w", err)
		}
		cert, key, err := parseCertKey(certPEM, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("parse ca: %w", err)
		}
		return &CA{CACert: cert, CAKey: key}, nil
	}

	// Generate new self-signed root CA.
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate CA key: %w", err)
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "SeaShell Root CA"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		MaxPathLen:            2,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, fmt.Errorf("create CA cert: %w", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}

	// Persist cert.
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		return nil, fmt.Errorf("write ca cert: %w", err)
	}

	// Persist key.
	keyDER, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("marshal CA key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, fmt.Errorf("write ca key: %w", err)
	}

	return &CA{CACert: cert, CAKey: privKey}, nil
}

// IssueNodeCert signs a CSR and returns the leaf certificate PEM.
// For regional nodes (tier="regional") the cert is marked as a CA with MaxPathLen=0,
// allowing it to sign branch leaf certs.
func (ca *CA) IssueNodeCert(csrPEM []byte, nodeTier, nodeID string) (certPEM []byte, err error) {
	block, _ := pem.Decode(csrPEM)
	if block == nil {
		return nil, fmt.Errorf("decode CSR PEM")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CSR: %w", err)
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("CSR signature invalid: %w", err)
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	isCA := nodeTier == "regional"
	maxPathLen := 0
	if !isCA {
		maxPathLen = -1
	}

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "SeaShell Node " + nodeID},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(5 * 365 * 24 * time.Hour),
		IsCA:                  isCA,
		MaxPathLen:            maxPathLen,
		MaxPathLenZero:        isCA,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		DNSNames:              csr.DNSNames,
		IPAddresses:           csr.IPAddresses,
		URIs:                  csr.URIs,
	}
	if isCA {
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.CACert, csr.PublicKey, ca.CAKey)
	if err != nil {
		return nil, fmt.Errorf("sign cert: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), nil
}

// CACertPEM returns the CA certificate as PEM bytes.
func (ca *CA) CACertPEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.CACert.Raw})
}

// GenerateNodeCSR generates a new ECDSA keypair and a CSR for the given nodeURL.
// Returns (privKeyPEM, csrPEM, certFingerprint, error).
func GenerateNodeCSR(nodeURL, nodeID string) (privKeyPEM, csrPEM []byte, err error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}

	// Add nodeURL as a URI SAN if valid.
	var uris []*url.URL
	if parsed, e := url.Parse(nodeURL); e == nil && parsed.Host != "" {
		uris = []*url.URL{parsed}
	}

	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: "SeaShell Node " + nodeID},
		URIs:    uris,
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create CSR: %w", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal key: %w", err)
	}

	privKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	csrPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})
	return privKeyPEM, csrPEM, nil
}

// CertFingerprint returns the SHA-256 hex fingerprint of a PEM-encoded certificate.
func CertFingerprint(certPEM []byte) (string, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return "", fmt.Errorf("decode cert PEM")
	}
	sum := sha256.Sum256(block.Bytes)
	return fmt.Sprintf("%x", sum), nil
}

// BuildServerTLSConfig returns a tls.Config for an mTLS server (requires client certs).
// certFile/keyFile are the server cert/key; caCertFile is the CA to verify client certs.
func BuildServerTLSConfig(certFile, keyFile, caCertFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load server cert: %w", err)
	}
	caPEM, err := os.ReadFile(caCertFile)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("parse CA cert")
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// parseCertKey parses PEM-encoded cert and EC private key.
func parseCertKey(certPEM, keyPEM []byte) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("decode cert PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("decode key PEM")
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}
