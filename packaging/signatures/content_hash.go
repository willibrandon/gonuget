package signatures

import (
	"archive/zip"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strings"
)

// GetPackageContentHash calculates the SHA512 hash of a signed package excluding the signature file.
// This matches NuGet.Client's SignedPackageArchiveUtility.GetPackageContentHash behavior.
// Reference: NuGet.Client SignedPackageArchiveUtility.cs GetPackageContentHash
func GetPackageContentHash(r io.ReadSeeker) (string, error) {
	// Seek to start
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek to start: %w", err)
	}

	// Read as ZIP archive to find signature file
	size, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return "", fmt.Errorf("get size: %w", err)
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek to start: %w", err)
	}

	// Open ZIP to check for signature
	zipReader, err := zip.NewReader(r.(io.ReaderAt), size)
	if err != nil {
		return "", fmt.Errorf("open zip: %w", err)
	}

	// Find signature file
	var signatureFile *zip.File
	for _, f := range zipReader.File {
		if strings.EqualFold(f.Name, ".signature.p7s") {
			signatureFile = f
			break
		}
	}

	if signatureFile == nil {
		// Not a signed package, return empty string to indicate unsigned
		return "", nil
	}

	// Read ZIP metadata for signed package
	metadata, err := readSignedArchiveMetadata(r)
	if err != nil {
		return "", fmt.Errorf("read archive metadata: %w", err)
	}

	// Calculate hash excluding signature
	hash := sha512.New()

	// Hash from start to beginning of file headers
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	if err := hashUntilPosition(r, hash, metadata.StartOfLocalFileHeaders); err != nil {
		return "", err
	}

	// Hash all file entries except signature
	entriesWithoutSig := removeSignatureAndSortByOffset(metadata)
	for _, entry := range entriesWithoutSig {
		if _, err := r.Seek(entry.OffsetToLocalFileHeader, io.SeekStart); err != nil {
			return "", err
		}
		if err := hashUntilPosition(r, hash, entry.OffsetToLocalFileHeader+entry.FileEntryTotalSize); err != nil {
			return "", err
		}
	}

	// Sort by position for central directory records
	sort.Slice(entriesWithoutSig, func(i, j int) bool {
		return entriesWithoutSig[i].Position < entriesWithoutSig[j].Position
	})

	// Hash central directory records with adjusted offsets
	for _, entry := range entriesWithoutSig {
		if _, err := r.Seek(entry.Position, io.SeekStart); err != nil {
			return "", err
		}

		// Hash up to relative offset field (42 bytes from start of central directory header)
		if err := hashUntilPosition(r, hash, entry.Position+42); err != nil {
			return "", err
		}

		// Read and adjust relative offset
		var relativeOffset uint32
		if err := binary.Read(r, binary.LittleEndian, &relativeOffset); err != nil {
			return "", err
		}
		adjustedOffset := uint32(int64(relativeOffset) + entry.ChangeInOffset)
		if err := binary.Write(hash, binary.LittleEndian, adjustedOffset); err != nil {
			return "", err
		}

		// Hash remaining header fields (filename, extra field, comment)
		// Current position + (HeaderSize - 46 fixed fields)
		currentPos, _ := r.Seek(0, io.SeekCurrent)
		remainingSize := entry.HeaderSize - 46 // 46 = fixed fields size
		if err := hashUntilPosition(r, hash, currentPos+remainingSize); err != nil {
			return "", err
		}
	}

	// Hash End of Central Directory Record with adjustments
	if _, err := r.Seek(metadata.EndOfCentralDirectory, io.SeekStart); err != nil {
		return "", err
	}

	// Hash first 8 bytes of EOCDR (signature + disk numbers)
	if err := hashUntilPosition(r, hash, metadata.EndOfCentralDirectory+8); err != nil {
		return "", err
	}

	// Read and adjust entry counts (subtract 1 for signature file)
	var totalEntries, totalEntriesOnDisk uint16
	if err := binary.Read(r, binary.LittleEndian, &totalEntries); err != nil {
		return "", err
	}
	if err := binary.Read(r, binary.LittleEndian, &totalEntriesOnDisk); err != nil {
		return "", err
	}
	if err := binary.Write(hash, binary.LittleEndian, totalEntries-1); err != nil {
		return "", err
	}
	if err := binary.Write(hash, binary.LittleEndian, totalEntriesOnDisk-1); err != nil {
		return "", err
	}

	// Read and adjust central directory size (subtract signature header size)
	var cdSize uint32
	if err := binary.Read(r, binary.LittleEndian, &cdSize); err != nil {
		return "", err
	}
	sigHeader := metadata.CentralDirectoryHeaders[metadata.SignatureCentralDirectoryHeaderIndex]
	adjustedCDSize := uint32(int64(cdSize) - sigHeader.HeaderSize)
	if err := binary.Write(hash, binary.LittleEndian, adjustedCDSize); err != nil {
		return "", err
	}

	// Read and adjust central directory offset (subtract signature file entry size)
	var cdOffset uint32
	if err := binary.Read(r, binary.LittleEndian, &cdOffset); err != nil {
		return "", err
	}
	adjustedCDOffset := uint32(int64(cdOffset) - sigHeader.FileEntryTotalSize)
	if err := binary.Write(hash, binary.LittleEndian, adjustedCDOffset); err != nil {
		return "", err
	}

	// Hash remaining EOCDR fields
	endSize, _ := r.Seek(0, io.SeekEnd)
	currentPos, _ := r.Seek(0, io.SeekCurrent)
	if err := hashUntilPosition(r, hash, endSize); err != nil && currentPos < endSize {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}

// SignedPackageArchiveMetadata holds metadata about a signed package archive
type SignedPackageArchiveMetadata struct {
	CentralDirectoryHeaders              []CentralDirectoryHeaderMetadata
	StartOfLocalFileHeaders              int64
	EndOfCentralDirectory                int64
	SignatureCentralDirectoryHeaderIndex int
}

// CentralDirectoryHeaderMetadata holds metadata about a central directory header
type CentralDirectoryHeaderMetadata struct {
	Position                int64
	OffsetToLocalFileHeader int64
	FileEntryTotalSize      int64
	IsPackageSignatureFile  bool
	HeaderSize              int64
	ChangeInOffset          int64
	IndexInHeaders          int
}

func readSignedArchiveMetadata(r io.ReadSeeker) (*SignedPackageArchiveMetadata, error) {
	size, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	metadata := &SignedPackageArchiveMetadata{
		StartOfLocalFileHeaders:              size,
		SignatureCentralDirectoryHeaderIndex: -1,
	}

	// Find End of Central Directory Record
	eocdr, eocdrOffset, err := findEndOfCentralDirectory(r)
	if err != nil {
		return nil, err
	}
	metadata.EndOfCentralDirectory = eocdrOffset

	// Seek to central directory
	if _, err := r.Seek(int64(eocdr.CentralDirectoryOffset), io.SeekStart); err != nil {
		return nil, err
	}

	// Read all central directory headers
	index := 0
	for {
		headerPos, _ := r.Seek(0, io.SeekCurrent)

		header, err := readCentralDirectoryHeader(r)
		if err != nil {
			break // End of central directory headers
		}

		isSignature := strings.EqualFold(header.FileName, ".signature.p7s")

		cdMetadata := CentralDirectoryHeaderMetadata{
			Position:                headerPos,
			OffsetToLocalFileHeader: int64(header.RelativeOffsetOfLocalHeader),
			HeaderSize:              int64(header.GetSizeInBytes()),
			IsPackageSignatureFile:  isSignature,
			IndexInHeaders:          index,
		}

		// Calculate file entry total size (local header + file data)
		if _, err := r.Seek(cdMetadata.OffsetToLocalFileHeader, io.SeekStart); err != nil {
			return nil, err
		}
		localHeader, err := readLocalFileHeader(r)
		if err != nil {
			return nil, err
		}
		cdMetadata.FileEntryTotalSize = int64(30 + uint32(localHeader.FileNameLength) + uint32(localHeader.ExtraFieldLength) + localHeader.CompressedSize)

		metadata.StartOfLocalFileHeaders = min(metadata.StartOfLocalFileHeaders, cdMetadata.OffsetToLocalFileHeader)

		if isSignature {
			metadata.SignatureCentralDirectoryHeaderIndex = index
		}

		metadata.CentralDirectoryHeaders = append(metadata.CentralDirectoryHeaders, cdMetadata)

		// Seek back to continue reading central directory
		if _, err := r.Seek(headerPos+cdMetadata.HeaderSize, io.SeekStart); err != nil {
			return nil, err
		}

		index++
	}

	return metadata, nil
}

func removeSignatureAndSortByOffset(metadata *SignedPackageArchiveMetadata) []CentralDirectoryHeaderMetadata {
	// Remove signature central directory record
	var result []CentralDirectoryHeaderMetadata
	for i, entry := range metadata.CentralDirectoryHeaders {
		if i != metadata.SignatureCentralDirectoryHeaderIndex {
			result = append(result, entry)
		}
	}

	// Sort by order of file entries (offset to local file header)
	sort.Slice(result, func(i, j int) bool {
		return result[i].OffsetToLocalFileHeader < result[j].OffsetToLocalFileHeader
	})

	// Update offsets with removed signature
	// Reference: NuGet.Client RemoveSignatureAndOrderByOffset lines 544-549
	var previousRecordFileEntryEnd int64
	for i := range result {
		entry := &result[i]
		entry.ChangeInOffset = previousRecordFileEntryEnd - entry.OffsetToLocalFileHeader
		previousRecordFileEntryEnd = entry.OffsetToLocalFileHeader + entry.FileEntryTotalSize + entry.ChangeInOffset
	}

	return result
}

func hashUntilPosition(r io.Reader, h io.Writer, endPos int64) error {
	if seeker, ok := r.(io.Seeker); ok {
		currentPos, _ := seeker.Seek(0, io.SeekCurrent)
		toRead := endPos - currentPos
		if toRead <= 0 {
			return nil
		}
		_, err := io.CopyN(h, r, toRead)
		return err
	}
	// Fallback for non-seekable
	_, err := io.Copy(h, r)
	return err
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// ZIP structures

type endOfCentralDirectory struct {
	Signature              uint32
	DiskNumber             uint16
	CentralDirectoryDisk   uint16
	NumEntriesOnDisk       uint16
	NumEntries             uint16
	CentralDirectorySize   uint32
	CentralDirectoryOffset uint32
	CommentLength          uint16
}

type centralDirectoryHeader struct {
	Signature                   uint32
	VersionMadeBy               uint16
	VersionNeededToExtract      uint16
	GeneralPurposeBitFlag       uint16
	CompressionMethod           uint16
	LastModFileTime             uint16
	LastModFileDate             uint16
	CRC32                       uint32
	CompressedSize              uint32
	UncompressedSize            uint32
	FileNameLength              uint16
	ExtraFieldLength            uint16
	FileCommentLength           uint16
	DiskNumberStart             uint16
	InternalFileAttributes      uint16
	ExternalFileAttributes      uint32
	RelativeOffsetOfLocalHeader uint32
	FileName                    string
}

func (h *centralDirectoryHeader) GetSizeInBytes() uint32 {
	return 46 + uint32(h.FileNameLength) + uint32(h.ExtraFieldLength) + uint32(h.FileCommentLength)
}

type localFileHeader struct {
	Signature              uint32
	VersionNeededToExtract uint16
	GeneralPurposeBitFlag  uint16
	CompressionMethod      uint16
	LastModFileTime        uint16
	LastModFileDate        uint16
	CRC32                  uint32
	CompressedSize         uint32
	UncompressedSize       uint32
	FileNameLength         uint16
	ExtraFieldLength       uint16
}

func findEndOfCentralDirectory(r io.ReadSeeker) (*endOfCentralDirectory, int64, error) {
	// EOCDR is at the end, search backwards
	size, _ := r.Seek(0, io.SeekEnd)

	// Search up to 64KB backwards for EOCDR signature
	searchSize := int64(65536)
	if size < searchSize {
		searchSize = size
	}

	buf := make([]byte, searchSize)
	if _, err := r.Seek(size-searchSize, io.SeekStart); err != nil {
		return nil, 0, err
	}
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, 0, err
	}

	// Search for EOCDR signature 0x06054b50
	for i := len(buf) - 22; i >= 0; i-- {
		if binary.LittleEndian.Uint32(buf[i:]) == 0x06054b50 {
			// Found EOCDR
			offset := size - searchSize + int64(i)
			if _, err := r.Seek(offset, io.SeekStart); err != nil {
				return nil, 0, err
			}

			eocdr := &endOfCentralDirectory{}
			if err := binary.Read(r, binary.LittleEndian, eocdr); err != nil {
				return nil, 0, err
			}

			return eocdr, offset, nil
		}
	}

	return nil, 0, fmt.Errorf("end of central directory not found")
}

func readCentralDirectoryHeader(r io.Reader) (*centralDirectoryHeader, error) {
	header := &centralDirectoryHeader{}

	if err := binary.Read(r, binary.LittleEndian, &header.Signature); err != nil {
		return nil, err
	}

	if header.Signature != 0x02014b50 {
		return nil, fmt.Errorf("invalid central directory header signature")
	}

	if err := binary.Read(r, binary.LittleEndian, &header.VersionMadeBy); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.VersionNeededToExtract); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.GeneralPurposeBitFlag); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.CompressionMethod); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.LastModFileTime); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.LastModFileDate); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.CRC32); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.CompressedSize); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.UncompressedSize); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.FileNameLength); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.ExtraFieldLength); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.FileCommentLength); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.DiskNumberStart); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.InternalFileAttributes); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.ExternalFileAttributes); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.RelativeOffsetOfLocalHeader); err != nil {
		return nil, err
	}

	// Read filename
	fileNameBytes := make([]byte, header.FileNameLength)
	if _, err := io.ReadFull(r, fileNameBytes); err != nil {
		return nil, err
	}
	header.FileName = string(fileNameBytes)

	// Skip extra field and comment
	skipSize := int64(header.ExtraFieldLength) + int64(header.FileCommentLength)
	if seeker, ok := r.(io.Seeker); ok {
		_, _ = seeker.Seek(skipSize, io.SeekCurrent)
	} else {
		_, _ = io.CopyN(io.Discard, r, skipSize)
	}

	return header, nil
}

func readLocalFileHeader(r io.Reader) (*localFileHeader, error) {
	header := &localFileHeader{}

	if err := binary.Read(r, binary.LittleEndian, &header.Signature); err != nil {
		return nil, err
	}

	if header.Signature != 0x04034b50 {
		return nil, fmt.Errorf("invalid local file header signature")
	}

	if err := binary.Read(r, binary.LittleEndian, &header.VersionNeededToExtract); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.GeneralPurposeBitFlag); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.CompressionMethod); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.LastModFileTime); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.LastModFileDate); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.CRC32); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.CompressedSize); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.UncompressedSize); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.FileNameLength); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.ExtraFieldLength); err != nil {
		return nil, err
	}

	return header, nil
}
