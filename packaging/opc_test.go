package packaging

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/willibrandon/gonuget/version"
)

func TestGenerateContentTypes(t *testing.T) {
	files := []PackageFile{
		{TargetPath: "lib/net6.0/test.dll"},
		{TargetPath: "lib/net6.0/test.xml"},
		{TargetPath: "content/readme.txt"},
		{TargetPath: "TestPackage.nuspec"},
	}

	contentTypes, err := GenerateContentTypes(files)
	if err != nil {
		t.Fatalf("GenerateContentTypes() error = %v", err)
	}

	if contentTypes.Xmlns != OPCContentTypesNamespace {
		t.Errorf("Xmlns = %q, want %q", contentTypes.Xmlns, OPCContentTypesNamespace)
	}

	// Check required defaults
	if len(contentTypes.Defaults) < 2 {
		t.Fatalf("len(Defaults) = %d, want at least 2", len(contentTypes.Defaults))
	}

	// Verify rels default
	foundRels := false
	foundPsmdcp := false
	for _, def := range contentTypes.Defaults {
		if def.Extension == "rels" && def.ContentType == RelationshipContentType {
			foundRels = true
		}
		if def.Extension == "psmdcp" && def.ContentType == CorePropertiesContentType {
			foundPsmdcp = true
		}
	}

	if !foundRels {
		t.Error("Required 'rels' default not found")
	}

	if !foundPsmdcp {
		t.Error("Required 'psmdcp' default not found")
	}

	// Check file extensions are added
	foundDll := false
	foundXML := false
	foundTxt := false
	foundNuspec := false

	for _, def := range contentTypes.Defaults {
		switch def.Extension {
		case "dll":
			foundDll = true
		case "xml":
			foundXML = true
		case "txt":
			foundTxt = true
		case "nuspec":
			foundNuspec = true
		}
	}

	if !foundDll {
		t.Error("Expected 'dll' extension default")
	}
	if !foundXML {
		t.Error("Expected 'xml' extension default")
	}
	if !foundTxt {
		t.Error("Expected 'txt' extension default")
	}
	if !foundNuspec {
		t.Error("Expected 'nuspec' extension default")
	}
}

func TestGenerateContentTypes_FilesWithoutExtension(t *testing.T) {
	files := []PackageFile{
		{TargetPath: "lib/net6.0/test.dll"},
		{TargetPath: "LICENSE"},
		{TargetPath: "README"},
	}

	contentTypes, err := GenerateContentTypes(files)
	if err != nil {
		t.Fatalf("GenerateContentTypes() error = %v", err)
	}

	// Check overrides for files without extensions
	if len(contentTypes.Overrides) != 2 {
		t.Fatalf("len(Overrides) = %d, want 2", len(contentTypes.Overrides))
	}

	// Overrides should be sorted
	if contentTypes.Overrides[0].PartName != "/LICENSE" {
		t.Errorf("Overrides[0].PartName = %q, want /LICENSE", contentTypes.Overrides[0].PartName)
	}

	if contentTypes.Overrides[1].PartName != "/README" {
		t.Errorf("Overrides[1].PartName = %q, want /README", contentTypes.Overrides[1].PartName)
	}

	// Check content type
	for _, override := range contentTypes.Overrides {
		if override.ContentType != DefaultContentType {
			t.Errorf("Override ContentType = %q, want %q", override.ContentType, DefaultContentType)
		}
	}
}

func TestGenerateContentTypes_EmptyFiles(t *testing.T) {
	files := []PackageFile{}

	contentTypes, err := GenerateContentTypes(files)
	if err != nil {
		t.Fatalf("GenerateContentTypes() error = %v", err)
	}

	// Should still have required defaults
	if len(contentTypes.Defaults) != 2 {
		t.Errorf("len(Defaults) = %d, want 2 (rels and psmdcp)", len(contentTypes.Defaults))
	}

	// Should have no overrides
	if len(contentTypes.Overrides) != 0 {
		t.Errorf("len(Overrides) = %d, want 0", len(contentTypes.Overrides))
	}
}

func TestGenerateContentTypes_DeterministicOutput(t *testing.T) {
	files := []PackageFile{
		{TargetPath: "z.zip"},
		{TargetPath: "a.txt"},
		{TargetPath: "m.dll"},
	}

	contentTypes, err := GenerateContentTypes(files)
	if err != nil {
		t.Fatalf("GenerateContentTypes() error = %v", err)
	}

	// Extensions should be sorted (after rels and psmdcp)
	// Expected order: rels, psmdcp, dll, txt, zip
	extensions := make([]string, 0, len(contentTypes.Defaults))
	for _, def := range contentTypes.Defaults {
		extensions = append(extensions, def.Extension)
	}

	// Check that file extensions are sorted
	fileExtensions := extensions[2:] // Skip rels and psmdcp
	for i := 1; i < len(fileExtensions); i++ {
		if fileExtensions[i-1] > fileExtensions[i] {
			t.Errorf("Extensions not sorted: %v", fileExtensions)
			break
		}
	}
}

func TestWriteContentTypes(t *testing.T) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	files := []PackageFile{
		{TargetPath: "lib/net6.0/test.dll"},
		{TargetPath: "LICENSE"},
	}

	err := WriteContentTypes(zipWriter, files)
	if err != nil {
		t.Fatalf("WriteContentTypes() error = %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close ZIP error = %v", err)
	}

	// Read ZIP and verify [Content_Types].xml exists
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Open ZIP error = %v", err)
	}

	var contentTypesFile *zip.File
	for _, file := range zipReader.File {
		if file.Name == OPCContentTypesPath {
			contentTypesFile = file
			break
		}
	}

	if contentTypesFile == nil {
		t.Fatal("Content_Types.xml not found in ZIP")
	}

	// Read and parse XML
	rc, err := contentTypesFile.Open()
	if err != nil {
		t.Fatalf("Open Content_Types.xml error = %v", err)
	}
	defer func() { _ = rc.Close() }()

	var contentTypes ContentTypesXML
	decoder := xml.NewDecoder(rc)
	if err := decoder.Decode(&contentTypes); err != nil {
		t.Fatalf("Decode XML error = %v", err)
	}

	// Verify structure
	if contentTypes.Xmlns != OPCContentTypesNamespace {
		t.Errorf("Xmlns = %q, want %q", contentTypes.Xmlns, OPCContentTypesNamespace)
	}
}

func TestGenerateRelationships(t *testing.T) {
	nuspecFileName := "TestPackage.nuspec"
	corePropsPath := "package/services/metadata/core-properties/test.psmdcp"

	rels := GenerateRelationships(nuspecFileName, corePropsPath)

	if rels.Xmlns != OPCRelationshipsNamespace {
		t.Errorf("Xmlns = %q, want %q", rels.Xmlns, OPCRelationshipsNamespace)
	}

	// Should have 2 relationships
	if len(rels.Relationships) != 2 {
		t.Fatalf("len(Relationships) = %d, want 2", len(rels.Relationships))
	}

	// Check manifest relationship
	manifestRel := rels.Relationships[0]
	if manifestRel.Type != OPCManifestRelType {
		t.Errorf("Manifest Type = %q, want %q", manifestRel.Type, OPCManifestRelType)
	}

	expectedTarget := "/" + nuspecFileName
	if manifestRel.Target != expectedTarget {
		t.Errorf("Manifest Target = %q, want %q", manifestRel.Target, expectedTarget)
	}

	if !strings.HasPrefix(manifestRel.ID, "R") {
		t.Errorf("Manifest ID = %q, should start with 'R'", manifestRel.ID)
	}

	// Check core properties relationship
	corePropsRel := rels.Relationships[1]
	expectedType := "http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties"
	if corePropsRel.Type != expectedType {
		t.Errorf("CoreProps Type = %q, want %q", corePropsRel.Type, expectedType)
	}

	expectedCorePropsTarget := "/" + corePropsPath
	if corePropsRel.Target != expectedCorePropsTarget {
		t.Errorf("CoreProps Target = %q, want %q", corePropsRel.Target, expectedCorePropsTarget)
	}
}

func TestGenerateRelationships_NoCoreProperties(t *testing.T) {
	nuspecFileName := "TestPackage.nuspec"

	rels := GenerateRelationships(nuspecFileName, "")

	// Should have only 1 relationship (manifest)
	if len(rels.Relationships) != 1 {
		t.Fatalf("len(Relationships) = %d, want 1", len(rels.Relationships))
	}

	manifestRel := rels.Relationships[0]
	if manifestRel.Type != OPCManifestRelType {
		t.Errorf("Type = %q, want %q", manifestRel.Type, OPCManifestRelType)
	}
}

func TestGenerateRelationshipID(t *testing.T) {
	id := GenerateRelationshipID()

	// IDs should start with 'R'
	if !strings.HasPrefix(id, "R") {
		t.Errorf("ID = %q, should start with 'R'", id)
	}

	// Should be hex after 'R'
	hexPart := id[1:]
	if len(hexPart) == 0 {
		t.Error("No hex part in ID")
	}

	// Verify it's valid hex
	for _, c := range hexPart {
		if (c < '0' || c > '9') && (c < 'A' || c > 'F') {
			t.Errorf("ID contains non-hex character: %c", c)
		}
	}
}

func TestWriteRelationships(t *testing.T) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	nuspecFileName := "TestPackage.nuspec"
	corePropsPath := "package/services/metadata/core-properties/test.psmdcp"

	err := WriteRelationships(zipWriter, nuspecFileName, corePropsPath)
	if err != nil {
		t.Fatalf("WriteRelationships() error = %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close ZIP error = %v", err)
	}

	// Read ZIP and verify _rels/.rels exists
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Open ZIP error = %v", err)
	}

	var relsFile *zip.File
	for _, file := range zipReader.File {
		if file.Name == OPCRelationshipsPath {
			relsFile = file
			break
		}
	}

	if relsFile == nil {
		t.Fatal("_rels/.rels not found in ZIP")
	}

	// Read and parse XML
	rc, err := relsFile.Open()
	if err != nil {
		t.Fatalf("Open _rels/.rels error = %v", err)
	}
	defer func() { _ = rc.Close() }()

	var rels RelationshipsXML
	decoder := xml.NewDecoder(rc)
	if err := decoder.Decode(&rels); err != nil {
		t.Fatalf("Decode XML error = %v", err)
	}

	// Verify structure
	if rels.Xmlns != OPCRelationshipsNamespace {
		t.Errorf("Xmlns = %q, want %q", rels.Xmlns, OPCRelationshipsNamespace)
	}

	if len(rels.Relationships) != 2 {
		t.Errorf("len(Relationships) = %d, want 2", len(rels.Relationships))
	}
}

func TestGenerateCoreProperties(t *testing.T) {
	ver := version.MustParse("1.2.3")
	metadata := PackageMetadata{
		ID:          "TestPackage",
		Version:     ver,
		Description: "Test description",
		Authors:     []string{"Author1", "Author2"},
		Tags:        []string{"tag1", "tag2", "tag3"},
	}

	props := GenerateCoreProperties(metadata)

	if props.XmlnsDC != DCNamespace {
		t.Errorf("XmlnsDC = %q, want %q", props.XmlnsDC, DCNamespace)
	}

	if props.XmlnsDCTerms != DCTermsNamespace {
		t.Errorf("XmlnsDCTerms = %q, want %q", props.XmlnsDCTerms, DCTermsNamespace)
	}

	if props.XmlnsXSI != XSINamespace {
		t.Errorf("XmlnsXSI = %q, want %q", props.XmlnsXSI, XSINamespace)
	}

	if props.Creator != "Author1, Author2" {
		t.Errorf("Creator = %q, want 'Author1, Author2'", props.Creator)
	}

	if props.Description != "Test description" {
		t.Errorf("Description = %q, want 'Test description'", props.Description)
	}

	if props.Identifier != "TestPackage" {
		t.Errorf("Identifier = %q, want TestPackage", props.Identifier)
	}

	if props.Version != "1.2.3" {
		t.Errorf("Version = %q, want 1.2.3", props.Version)
	}

	if props.Keywords != "tag1 tag2 tag3" {
		t.Errorf("Keywords = %q, want 'tag1 tag2 tag3'", props.Keywords)
	}

	if props.LastModifiedBy != "gonuget" {
		t.Errorf("LastModifiedBy = %q, want gonuget", props.LastModifiedBy)
	}
}

func TestGenerateCoreProperties_MinimalMetadata(t *testing.T) {
	metadata := PackageMetadata{
		ID:          "MinimalPackage",
		Description: "Minimal",
	}

	props := GenerateCoreProperties(metadata)

	if props.Identifier != "MinimalPackage" {
		t.Errorf("Identifier = %q, want MinimalPackage", props.Identifier)
	}

	if props.Description != "Minimal" {
		t.Errorf("Description = %q, want Minimal", props.Description)
	}

	// Empty fields
	if props.Creator != "" {
		t.Errorf("Creator = %q, want empty", props.Creator)
	}

	if props.Version != "" {
		t.Errorf("Version = %q, want empty", props.Version)
	}

	if props.Keywords != "" {
		t.Errorf("Keywords = %q, want empty", props.Keywords)
	}
}

func TestWriteCoreProperties(t *testing.T) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	ver := version.MustParse("1.0.0")
	metadata := PackageMetadata{
		ID:          "TestPackage",
		Version:     ver,
		Description: "Test",
		Authors:     []string{"Test Author"},
	}

	filename, err := WriteCoreProperties(zipWriter, metadata)
	if err != nil {
		t.Fatalf("WriteCoreProperties() error = %v", err)
	}

	if !strings.HasPrefix(filename, OPCCorePropertiesPath) {
		t.Errorf("filename = %q, should start with %q", filename, OPCCorePropertiesPath)
	}

	if !strings.HasSuffix(filename, ".psmdcp") {
		t.Errorf("filename = %q, should end with .psmdcp", filename)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close ZIP error = %v", err)
	}

	// Read ZIP and verify file exists
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Open ZIP error = %v", err)
	}

	var psmdcpFile *zip.File
	for _, file := range zipReader.File {
		if file.Name == filename {
			psmdcpFile = file
			break
		}
	}

	if psmdcpFile == nil {
		t.Fatalf("Core properties file %q not found in ZIP", filename)
	}

	// Verify directory structure exists
	foundPackage := false
	foundServices := false
	foundMetadata := false
	foundCoreProps := false

	for _, file := range zipReader.File {
		switch file.Name {
		case "package/":
			foundPackage = true
		case "package/services/":
			foundServices = true
		case "package/services/metadata/":
			foundMetadata = true
		case OPCCorePropertiesPath:
			foundCoreProps = true
		}
	}

	if !foundPackage {
		t.Error("package/ directory not found")
	}
	if !foundServices {
		t.Error("package/services/ directory not found")
	}
	if !foundMetadata {
		t.Error("package/services/metadata/ directory not found")
	}
	if !foundCoreProps {
		t.Error("package/services/metadata/core-properties/ directory not found")
	}

	// Read and parse XML
	rc, err := psmdcpFile.Open()
	if err != nil {
		t.Fatalf("Open core properties error = %v", err)
	}
	defer func() { _ = rc.Close() }()

	var props CorePropertiesXML
	decoder := xml.NewDecoder(rc)
	if err := decoder.Decode(&props); err != nil {
		t.Fatalf("Decode XML error = %v", err)
	}

	// Verify content
	if props.Identifier != "TestPackage" {
		t.Errorf("Identifier = %q, want TestPackage", props.Identifier)
	}
}

func TestOPCXMLStructure(t *testing.T) {
	// Test that XML structures marshal correctly
	t.Run("ContentTypesXML", func(t *testing.T) {
		ct := &ContentTypesXML{
			Xmlns: OPCContentTypesNamespace,
			Defaults: []ContentTypeDefault{
				{Extension: "rels", ContentType: RelationshipContentType},
			},
			Overrides: []ContentTypeOverride{
				{PartName: "/LICENSE", ContentType: DefaultContentType},
			},
		}

		data, err := xml.MarshalIndent(ct, "", "  ")
		if err != nil {
			t.Fatalf("Marshal error = %v", err)
		}

		xmlStr := string(data)
		if !strings.Contains(xmlStr, "Types") {
			t.Error("XML should contain 'Types' element")
		}
		if !strings.Contains(xmlStr, OPCContentTypesNamespace) {
			t.Error("XML should contain namespace")
		}
	})

	t.Run("RelationshipsXML", func(t *testing.T) {
		rels := &RelationshipsXML{
			Xmlns: OPCRelationshipsNamespace,
			Relationships: []Relationship{
				{Type: OPCManifestRelType, Target: "/test.nuspec", ID: "R123"},
			},
		}

		data, err := xml.MarshalIndent(rels, "", "  ")
		if err != nil {
			t.Fatalf("Marshal error = %v", err)
		}

		xmlStr := string(data)
		if !strings.Contains(xmlStr, "Relationships") {
			t.Error("XML should contain 'Relationships' element")
		}
		if !strings.Contains(xmlStr, OPCRelationshipsNamespace) {
			t.Error("XML should contain namespace")
		}
	})

	t.Run("CorePropertiesXML", func(t *testing.T) {
		props := &CorePropertiesXML{
			XmlnsDC:      DCNamespace,
			XmlnsDCTerms: DCTermsNamespace,
			XmlnsXSI:     XSINamespace,
			Identifier:   "Test",
			Version:      "1.0.0",
		}

		data, err := xml.MarshalIndent(props, "", "  ")
		if err != nil {
			t.Fatalf("Marshal error = %v", err)
		}

		xmlStr := string(data)
		if !strings.Contains(xmlStr, "coreProperties") {
			t.Error("XML should contain 'coreProperties' element")
		}
		if !strings.Contains(xmlStr, OPCCorePropertiesNamespace) {
			t.Error("XML should contain namespace")
		}
	})
}
