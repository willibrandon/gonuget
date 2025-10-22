package signatures

import (
	"bytes"
	"crypto/rand"
	"encoding/asn1"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"
)

// TimestampClient provides RFC 3161 Time-Stamp Protocol (TSP) functionality.
// It sends timestamp requests to a timestamp authority (TSA) and validates responses.
// Timestamp tokens provide trusted proof-of-existence for signatures at a specific time.
type TimestampClient struct {
	url     string
	timeout time.Duration
	client  *http.Client
}

// NewTimestampClient creates a new RFC 3161 timestamp client for the specified TSA.
// The url parameter should point to an RFC 3161-compliant timestamp authority endpoint.
// The timeout applies to HTTP requests to the TSA.
func NewTimestampClient(url string, timeout time.Duration) *TimestampClient {
	return &TimestampClient{
		url:     url,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// RequestTimestamp requests an RFC 3161 timestamp token from the timestamp authority.
// It creates a TimeStampReq with the message hash and a cryptographic nonce, sends it
// to the TSA via HTTP POST, validates the response status and nonce, and returns the
// timestamp token (a SignedData ContentInfo structure).
// The messageHash should be the hash of the data to be timestamped (typically a signature).
// Returns the DER-encoded timestamp token ready to be added as an unsigned attribute.
func (c *TimestampClient) RequestTimestamp(messageHash []byte, hashAlg HashAlgorithmName) ([]byte, error) {
	// Generate nonce (32 bytes random, ensure valid per NuGet.Client)
	nonce, err := generateNonce()
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// Build TimeStampReq
	req, err := buildTimestampRequest(messageHash, hashAlg, nonce)
	if err != nil {
		return nil, fmt.Errorf("build timestamp request: %w", err)
	}

	// Encode request to DER
	reqBytes, err := asn1.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal timestamp request: %w", err)
	}

	// Send HTTP POST request
	httpReq, err := http.NewRequest("POST", c.url, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/timestamp-query")

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send timestamp request: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("timestamp server error: HTTP %d %s", httpResp.StatusCode, httpResp.Status)
	}

	// Read response body
	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read timestamp response: %w", err)
	}

	// Parse TimeStampResp
	var resp timestampResponse
	if _, err := asn1.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal timestamp response: %w", err)
	}

	// Verify status
	if resp.Status.Status != 0 && resp.Status.Status != 1 {
		return nil, fmt.Errorf("timestamp request rejected: status=%d", resp.Status.Status)
	}

	// Verify response contains token
	if len(resp.TimeStampToken.FullBytes) == 0 {
		return nil, fmt.Errorf("timestamp response missing token")
	}

	// Verify nonce matches (replay attack prevention)
	if err := verifyTimestampResponse(resp.TimeStampToken.FullBytes, messageHash, nonce); err != nil {
		return nil, fmt.Errorf("verify timestamp response: %w", err)
	}

	// Return the timestamp token (ContentInfo containing SignedData)
	return resp.TimeStampToken.FullBytes, nil
}

// RFC 3161 ASN.1 structures

type timestampRequest struct {
	Version        int
	MessageImprint messageImprint
	ReqPolicy      asn1.ObjectIdentifier `asn1:"optional"`
	Nonce          *big.Int              `asn1:"optional"`
	CertReq        bool                  `asn1:"optional,default:false"`
	Extensions     asn1.RawValue         `asn1:"optional,tag:0"`
}

type messageImprint struct {
	HashAlgorithm AlgorithmIdentifier
	HashedMessage []byte
}

type timestampResponse struct {
	Status         pkiStatusInfo
	TimeStampToken asn1.RawValue `asn1:"optional"`
}

type pkiStatusInfo struct {
	Status       int
	StatusString []string       `asn1:"optional"`
	FailInfo     asn1.BitString `asn1:"optional"`
}

// buildTimestampRequest creates an RFC 3161 TimeStampReq structure.
// It constructs a version 1 request containing the message imprint (hash algorithm + hash),
// a nonce for replay attack prevention, and requests the TSA certificate to be included.
// Returns a timestampRequest ready to be DER-encoded and sent to the TSA.
func buildTimestampRequest(messageHash []byte, hashAlg HashAlgorithmName, nonce []byte) (timestampRequest, error) {
	// Get hash algorithm OID
	hashAlgOID := getDigestAlgorithmOID(hashAlg)

	// Build MessageImprint
	mi := messageImprint{
		HashAlgorithm: AlgorithmIdentifier{
			Algorithm: hashAlgOID,
		},
		HashedMessage: messageHash,
	}

	// Convert nonce bytes to big.Int (big-endian)
	var nonceInt *big.Int
	if len(nonce) > 0 {
		nonceInt = new(big.Int).SetBytes(nonce)
	}

	// Build TimeStampReq
	req := timestampRequest{
		Version:        1,
		MessageImprint: mi,
		Nonce:          nonceInt,
		CertReq:        true, // Request TSA certificate
	}

	return req, nil
}

// generateNonce generates a 32-byte nonce for timestamp requests.
// Matches NuGet.Client's nonce generation (EnsureValidNonce).
func generateNonce() ([]byte, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Per NuGet.Client: ensure nonce is unsigned big-endian integer
	// Clear sign bit on most significant byte
	nonce[0] &= 0x7f

	return nonce, nil
}

// verifyTimestampResponse validates that a timestamp token matches the request.
// It parses the timestamp token (ContentInfo/SignedData structure), extracts the TSTInfo,
// and verifies that the message imprint hash and nonce match the expected values.
// This prevents replay attacks and ensures the timestamp applies to the correct data.
// Returns an error if verification fails.
func verifyTimestampResponse(tokenBytes, expectedHash, expectedNonce []byte) error {
	// Parse the timestamp token (it's a ContentInfo containing SignedData)
	var contentInfo ContentInfo
	if _, err := asn1.Unmarshal(tokenBytes, &contentInfo); err != nil {
		return fmt.Errorf("unmarshal content info: %w", err)
	}

	// Verify contentType is SignedData
	if !contentInfo.ContentType.Equal(oidSignedData) {
		return fmt.Errorf("invalid content type: expected SignedData")
	}

	// Parse SignedData
	var signedData SignedData
	if _, err := asn1.Unmarshal(contentInfo.Content.Bytes, &signedData); err != nil {
		return fmt.Errorf("unmarshal signed data: %w", err)
	}

	// Parse TSTInfo from eContent (eContent is OCTET STRING containing DER-encoded TSTInfo)
	var eContent []byte
	if _, err := asn1.Unmarshal(signedData.ContentInfo.Content.Bytes, &eContent); err != nil {
		return fmt.Errorf("unmarshal eContent: %w", err)
	}

	var tstInfo tstInfo
	if _, err := asn1.Unmarshal(eContent, &tstInfo); err != nil {
		return fmt.Errorf("unmarshal TSTInfo: %w", err)
	}

	// Verify message imprint hash matches
	if !bytes.Equal(tstInfo.MessageImprint.HashedMessage, expectedHash) {
		return fmt.Errorf("timestamp message imprint mismatch")
	}

	// Verify nonce matches (if present)
	if tstInfo.Nonce != nil {
		expectedNonceInt := new(big.Int).SetBytes(expectedNonce)
		if tstInfo.Nonce.Cmp(expectedNonceInt) != 0 {
			return fmt.Errorf("timestamp nonce mismatch")
		}
	}

	return nil
}

// tstInfo represents the RFC 3161 TSTInfo (Time-Stamp Token Info) structure.
// This is the signed content of a timestamp token, containing the timestamp generation time,
// message imprint, serial number, policy OID, and optional accuracy and nonce fields.
type tstInfo struct {
	Version        int
	Policy         asn1.ObjectIdentifier
	MessageImprint messageImprint
	SerialNumber   *big.Int
	GenTime        time.Time
	Accuracy       accuracy      `asn1:"optional"`
	Ordering       bool          `asn1:"optional,default:false"`
	Nonce          *big.Int      `asn1:"optional"`
	TSA            asn1.RawValue `asn1:"optional,tag:0"`
	Extensions     asn1.RawValue `asn1:"optional,tag:1"`
}

type accuracy struct {
	Seconds int `asn1:"optional"`
	Millis  int `asn1:"optional,tag:0"`
	Micros  int `asn1:"optional,tag:1"`
}
