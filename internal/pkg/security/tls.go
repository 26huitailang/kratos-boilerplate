package security

import (
	"crypto/tls"
	"fmt"
	"os"
)

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled      bool     `yaml:"enabled"`
	CertFile     string   `yaml:"cert_file"`
	KeyFile      string   `yaml:"key_file"`
	MinVersion   string   `yaml:"min_version"`
	CipherSuites []string `yaml:"cipher_suites"`
}

// DefaultTLSConfig returns default TLS configuration
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		Enabled:    false,
		CertFile:   "",
		KeyFile:    "",
		MinVersion: "1.2",
		CipherSuites: []string{
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		},
	}
}

// LoadTLSConfig loads TLS configuration from environment variables
func LoadTLSConfig() *TLSConfig {
	config := DefaultTLSConfig()

	if os.Getenv("TLS_ENABLED") == "true" {
		config.Enabled = true
	}

	if certFile := os.Getenv("TLS_CERT_FILE"); certFile != "" {
		config.CertFile = certFile
	}

	if keyFile := os.Getenv("TLS_KEY_FILE"); keyFile != "" {
		config.KeyFile = keyFile
	}

	if minVersion := os.Getenv("TLS_MIN_VERSION"); minVersion != "" {
		config.MinVersion = minVersion
	}

	return config
}

// GetTLSConfig returns a tls.Config based on the configuration
func (c *TLSConfig) GetTLSConfig() (*tls.Config, error) {
	if !c.Enabled {
		return nil, nil
	}

	if c.CertFile == "" || c.KeyFile == "" {
		return nil, fmt.Errorf("TLS enabled but cert_file or key_file not specified")
	}

	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   c.getMinVersion(),
		CipherSuites: c.getCipherSuites(),
		// Prefer server cipher suites
		PreferServerCipherSuites: true,
	}

	return tlsConfig, nil
}

// getMinVersion returns the TLS minimum version
func (c *TLSConfig) getMinVersion() uint16 {
	switch c.MinVersion {
	case "1.0":
		return tls.VersionTLS10
	case "1.1":
		return tls.VersionTLS11
	case "1.2":
		return tls.VersionTLS12
	case "1.3":
		return tls.VersionTLS13
	default:
		return tls.VersionTLS12 // Default to TLS 1.2
	}
}

// getCipherSuites returns the configured cipher suites
func (c *TLSConfig) getCipherSuites() []uint16 {
	var suites []uint16

	cipherMap := map[string]uint16{
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":    tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":  tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	}

	for _, suite := range c.CipherSuites {
		if cipherSuite, ok := cipherMap[suite]; ok {
			suites = append(suites, cipherSuite)
		}
	}

	// If no valid cipher suites configured, use secure defaults
	if len(suites) == 0 {
		suites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		}
	}

	return suites
}
