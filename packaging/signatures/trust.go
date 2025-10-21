package signatures

import (
	"crypto/x509"
	"fmt"
)

// TrustStore manages trusted root certificates
type TrustStore struct {
	roots *x509.CertPool
}

// NewTrustStore creates a new trust store
func NewTrustStore() *TrustStore {
	return &TrustStore{
		roots: x509.NewCertPool(),
	}
}

// NewTrustStoreFromSystem creates a trust store from system roots
func NewTrustStoreFromSystem() (*TrustStore, error) {
	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	return &TrustStore{
		roots: roots,
	}, nil
}

// AddCertificate adds a trusted root certificate
func (ts *TrustStore) AddCertificate(cert *x509.Certificate) {
	ts.roots.AddCert(cert)
}

// AddCertificatePEM adds a trusted root from PEM data
func (ts *TrustStore) AddCertificatePEM(pemData []byte) error {
	if !ts.roots.AppendCertsFromPEM(pemData) {
		return fmt.Errorf("failed to parse PEM certificate")
	}
	return nil
}

// GetRootPool returns the certificate pool
func (ts *TrustStore) GetRootPool() *x509.CertPool {
	return ts.roots
}
