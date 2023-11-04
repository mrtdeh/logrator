package tls_config

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

var config *tls.Config

type Config struct {
	CAPath     string
	CertPath   string
	KeyPath    string
	SkipVerify bool
}

func LoadTLSCredentials(opt Config) (*tls.Config, error) {

	ca := opt.CAPath
	cert := opt.CertPath
	key := opt.KeyPath

	// return nil if enabled skip verify tls
	if opt.SkipVerify {
		return nil, nil
	}
	// return tls config if not nil
	if config != nil {
		return config, nil
	}
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := os.ReadFile(ca)
	if err != nil {
		return nil, fmt.Errorf("path %s - %s", ca, err.Error())
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}
	clientCert, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	// Create the credentials and return it
	config = &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      certPool,
		// MinVersion:            tls.VersionTLS12,
		InsecureSkipVerify: true,
		// VerifyPeerCertificate: verifyPeerCertFunc(certPool),
		CipherSuites: []uint16{
			// tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			// tls.TLS_AES_256_GCM_SHA384,
			// tls.TLS_AES_128_GCM_SHA256,
			// tls.TLS_CHACHA20_POLY1305_SHA256,
			// tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			// tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			// tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			// tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
		CurvePreferences: []tls.CurveID{
			// tls.CurveP256,
			tls.CurveP384,
			tls.CurveP521,
		},
	}

	return config, nil
}

// verifyPeerCertFunc returns a function that verifies the peer certificate is
// in the cert pool.
func verifyPeerCertFunc(pool *x509.CertPool) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return errors.New("no certificates available to verify")
		}

		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return err
		}

		opts := x509.VerifyOptions{Roots: pool}
		if _, err = cert.Verify(opts); err != nil {
			return err
		}
		return nil
	}
}
