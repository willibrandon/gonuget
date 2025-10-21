# PKCS#7 / CMS ASN.1 Structures for NuGet Signatures

Based on RFC 5652 (Cryptographic Message Syntax)

## Core Structures

### ContentInfo (Outer wrapper)
```asn1
ContentInfo ::= SEQUENCE {
  contentType ContentType,
  content [0] EXPLICIT ANY DEFINED BY contentType }

ContentType ::= OBJECT IDENTIFIER
```

For SignedData, contentType = 1.2.840.113549.1.7.2

### SignedData
```asn1
SignedData ::= SEQUENCE {
  version CMSVersion,
  digestAlgorithms DigestAlgorithmIdentifiers,
  encapContentInfo EncapsulatedContentInfo,
  certificates [0] IMPLICIT CertificateSet OPTIONAL,
  crls [1] IMPLICIT RevocationInfoChoices OPTIONAL,
  signerInfos SignerInfos }

DigestAlgorithmIdentifiers ::= SET OF DigestAlgorithmIdentifier
SignerInfos ::= SET OF SignerInfo
CMSVersion ::= INTEGER  { v0(0), v1(1), v2(2), v3(3), v4(4), v5(5) }
```

### EncapsulatedContentInfo
```asn1
EncapsulatedContentInfo ::= SEQUENCE {
  eContentType ContentType,
  eContent [0] EXPLICIT OCTET STRING OPTIONAL }
```

### SignerInfo
```asn1
SignerInfo ::= SEQUENCE {
  version CMSVersion,
  sid SignerIdentifier,
  digestAlgorithm DigestAlgorithmIdentifier,
  signedAttrs [0] IMPLICIT SignedAttributes OPTIONAL,
  signatureAlgorithm SignatureAlgorithmIdentifier,
  signature SignatureValue,
  unsignedAttrs [1] IMPLICIT UnsignedAttributes OPTIONAL }

SignerIdentifier ::= CHOICE {
  issuerAndSerialNumber IssuerAndSerialNumber,
  subjectKeyIdentifier [0] SubjectKeyIdentifier }

IssuerAndSerialNumber ::= SEQUENCE {
  issuer Name,
  serialNumber CertificateSerialNumber }
```

### Attributes
```asn1
SignedAttributes ::= SET SIZE (1..MAX) OF Attribute
UnsignedAttributes ::= SET SIZE (1..MAX) OF Attribute

Attribute ::= SEQUENCE {
  attrType OBJECT IDENTIFIER,
  attrValues SET OF AttributeValue }

AttributeValue ::= ANY
```

### Algorithm Identifiers
```asn1
DigestAlgorithmIdentifier ::= AlgorithmIdentifier
SignatureAlgorithmIdentifier ::= AlgorithmIdentifier

AlgorithmIdentifier ::= SEQUENCE {
  algorithm OBJECT IDENTIFIER,
  parameters ANY DEFINED BY algorithm OPTIONAL }
```

## NuGet-Specific OIDs

### Signature Types (in SignedAttributes)
- Commitment Type Indication: 1.3.6.1.4.1.311.2.4.1
  - Author: 1.3.6.1.4.1.311.2.4.1.1
  - Repository: 1.3.6.1.4.1.311.2.4.1.2

### RFC 3161 Timestamp (in UnsignedAttributes)
- Timestamp Token: 1.2.840.113549.1.9.16.2.14

### Hash Algorithms
- SHA256: 2.16.840.1.101.3.4.2.1
- SHA384: 2.16.840.1.101.3.4.2.2
- SHA512: 2.16.840.1.101.3.4.2.3

## Go Implementation Notes

### ASN.1 Tags
- `[0] IMPLICIT` → `asn1:"optional,tag:0"`
- `[0] EXPLICIT` → `asn1:"explicit,optional,tag:0"`
- `[1] IMPLICIT` → `asn1:"optional,tag:1"`
- `SET OF` → `asn1:"set"`
- `SEQUENCE` → default, no tag needed

### Type Mappings
- `OBJECT IDENTIFIER` → `asn1.ObjectIdentifier`
- `INTEGER` → `int` or `*big.Int`
- `OCTET STRING` → `[]byte`
- `Name` (X.509) → `asn1.RawValue` (parse later with x509)
- `CHOICE` → Use separate parsing logic

## RFC 3161 Timestamp Structures

From RFC 3161 (Time-Stamp Protocol):

```asn1
TimeStampToken ::= ContentInfo
  -- contentType is id-signedData (1.2.840.113549.1.7.2)
  -- content is SignedData

TSTInfo ::= SEQUENCE {
  version INTEGER  { v1(1) },
  policy TSAPolicyId,
  messageImprint MessageImprint,
  serialNumber INTEGER,
  genTime GeneralizedTime,
  accuracy Accuracy OPTIONAL,
  ordering BOOLEAN DEFAULT FALSE,
  nonce INTEGER OPTIONAL,
  tsa [0] IMPLICIT GeneralName OPTIONAL,
  extensions [1] IMPLICIT Extensions OPTIONAL }

MessageImprint ::= SEQUENCE {
  hashAlgorithm AlgorithmIdentifier,
  hashedMessage OCTET STRING }
```

## References
- RFC 5652: https://www.rfc-editor.org/rfc/rfc5652.txt
- RFC 3161: https://www.rfc-editor.org/rfc/rfc3161.txt
