package mfsbsd

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const iso9660SectorSize = 2048

// FileISOExtractor extracts ISO9660/Rock Ridge payloads without mounting them.
type FileISOExtractor struct{}

type isoDirectoryRecord struct {
	name    string
	extent  uint32
	size    uint32
	mode    fs.FileMode
	dir     bool
	special bool
}

type rockRidgeInfo struct {
	name    string
	mode    fs.FileMode
	symlink string
}

// Extract reads isoPath directly and copies its ISO9660 tree to dest.
func (FileISOExtractor) Extract(ctx context.Context, isoPath string, dest string) error {
	if strings.TrimSpace(isoPath) == "" {
		return errors.New("ISO path is required")
	}
	if strings.TrimSpace(dest) == "" {
		return errors.New("extract destination is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	file, err := os.Open(isoPath)
	if err != nil {
		return fmt.Errorf("open ISO: %w", err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat ISO: %w", err)
	}
	root, err := readPrimaryVolumeRoot(file, info.Size())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("create extract destination: %w", err)
	}
	return extractISORecord(ctx, file, info.Size(), root, dest)
}

func readPrimaryVolumeRoot(reader io.ReaderAt, size int64) (isoDirectoryRecord, error) {
	for offset := int64(16 * iso9660SectorSize); offset+iso9660SectorSize <= size; offset += iso9660SectorSize {
		descriptor := make([]byte, iso9660SectorSize)
		if _, err := reader.ReadAt(descriptor, offset); err != nil {
			return isoDirectoryRecord{}, fmt.Errorf("read ISO volume descriptor: %w", err)
		}
		if string(descriptor[1:6]) != "CD001" {
			return isoDirectoryRecord{}, errors.New("ISO volume descriptor missing CD001 signature")
		}
		switch descriptor[0] {
		case 1:
			root, err := parseISODirectoryRecord(descriptor[156:])
			if err != nil {
				return isoDirectoryRecord{}, fmt.Errorf("parse ISO root directory record: %w", err)
			}
			if !root.dir {
				return isoDirectoryRecord{}, errors.New("ISO root directory record is not a directory")
			}
			return root, nil
		case 255:
			return isoDirectoryRecord{}, errors.New("ISO primary volume descriptor not found")
		}
	}
	return isoDirectoryRecord{}, errors.New("ISO primary volume descriptor not found")
}

func extractISORecord(ctx context.Context, reader io.ReaderAt, imageSize int64, record isoDirectoryRecord, dest string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateISOExtent(record, imageSize); err != nil {
		return err
	}
	if record.dir {
		mode := record.mode.Perm()
		if mode == 0 {
			mode = 0o755
		}
		if err := os.MkdirAll(dest, mode); err != nil {
			return fmt.Errorf("create ISO directory %s: %w", dest, err)
		}
		records, err := readISODirectory(ctx, reader, imageSize, record)
		if err != nil {
			return err
		}
		for _, child := range records {
			name, err := safeISOPathComponent(child.name)
			if err != nil {
				return err
			}
			if err := extractISORecord(ctx, reader, imageSize, child, filepath.Join(dest, name)); err != nil {
				return err
			}
		}
		return os.Chmod(dest, mode)
	}

	mode := record.mode.Perm()
	if mode == 0 {
		mode = 0o644
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("create ISO file parent %s: %w", dest, err)
	}
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create ISO file %s: %w", dest, err)
	}
	_, copyErr := io.Copy(out, io.NewSectionReader(reader, int64(record.extent)*iso9660SectorSize, int64(record.size)))
	closeErr := out.Close()
	if copyErr != nil {
		return fmt.Errorf("copy ISO file %s: %w", dest, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close ISO file %s: %w", dest, closeErr)
	}
	return os.Chmod(dest, mode)
}

func readISODirectory(ctx context.Context, reader io.ReaderAt, imageSize int64, record isoDirectoryRecord) ([]isoDirectoryRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateISOExtent(record, imageSize); err != nil {
		return nil, err
	}
	data := make([]byte, record.size)
	if _, err := reader.ReadAt(data, int64(record.extent)*iso9660SectorSize); err != nil {
		return nil, fmt.Errorf("read ISO directory %s: %w", record.name, err)
	}

	var records []isoDirectoryRecord
	for offset := 0; offset < len(data); {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		recordLen := int(data[offset])
		if recordLen == 0 {
			offset = ((offset / iso9660SectorSize) + 1) * iso9660SectorSize
			continue
		}
		if offset+recordLen > len(data) {
			return nil, fmt.Errorf("ISO directory record at offset %d exceeds directory size", offset)
		}
		child, err := parseISODirectoryRecord(data[offset : offset+recordLen])
		if err != nil {
			return nil, err
		}
		if !child.special {
			records = append(records, child)
		}
		offset += recordLen
	}
	return records, nil
}

func validateISOExtent(record isoDirectoryRecord, imageSize int64) error {
	if imageSize < 0 {
		return errors.New("ISO size is negative")
	}
	start := int64(record.extent) * iso9660SectorSize
	end := start + int64(record.size)
	if start > imageSize || end > imageSize {
		return fmt.Errorf("ISO record %q extent %d size %d exceeds ISO size %d", record.name, record.extent, record.size, imageSize)
	}
	return nil
}

func parseISODirectoryRecord(data []byte) (isoDirectoryRecord, error) {
	if len(data) == 0 {
		return isoDirectoryRecord{}, errors.New("empty ISO directory record")
	}
	recordLen := int(data[0])
	if recordLen == 0 {
		return isoDirectoryRecord{special: true}, nil
	}
	if recordLen < 34 {
		return isoDirectoryRecord{}, fmt.Errorf("ISO directory record length %d is too short", recordLen)
	}
	if recordLen > len(data) {
		return isoDirectoryRecord{}, fmt.Errorf("ISO directory record length %d exceeds buffer length %d", recordLen, len(data))
	}
	record := data[:recordLen]
	fileIDLen := int(record[32])
	fileIDEnd := 33 + fileIDLen
	if fileIDLen == 0 || fileIDEnd > len(record) {
		return isoDirectoryRecord{}, fmt.Errorf("invalid ISO file identifier length %d", fileIDLen)
	}

	fileID := string(record[33:fileIDEnd])
	special := fileID == "\x00" || fileID == "\x01"
	name := normalizeISO9660Name(fileID)
	systemUseStart := fileIDEnd
	if fileIDLen%2 == 0 {
		systemUseStart++
	}
	if systemUseStart > len(record) {
		systemUseStart = len(record)
	}
	rr, err := parseRockRidge(record[systemUseStart:])
	if err != nil {
		return isoDirectoryRecord{}, err
	}
	if rr.name != "" {
		name = rr.name
	}
	if rr.symlink != "" {
		return isoDirectoryRecord{}, fmt.Errorf("rock ridge symlink %q is not supported", name)
	}
	if record[25]&0x80 != 0 {
		return isoDirectoryRecord{}, fmt.Errorf("multi-extent ISO file %q is not supported", name)
	}

	return isoDirectoryRecord{
		name:    name,
		extent:  binary.LittleEndian.Uint32(record[2:6]),
		size:    binary.LittleEndian.Uint32(record[10:14]),
		mode:    rr.mode,
		dir:     record[25]&0x02 != 0,
		special: special,
	}, nil
}

func parseRockRidge(systemUse []byte) (rockRidgeInfo, error) {
	var info rockRidgeInfo
	for len(systemUse) >= 4 {
		length := int(systemUse[2])
		if length == 0 {
			break
		}
		if length < 4 || length > len(systemUse) {
			return rockRidgeInfo{}, fmt.Errorf("invalid Rock Ridge entry length %d", length)
		}
		entry := systemUse[:length]
		switch string(entry[0:2]) {
		case "NM":
			if length < 5 {
				return rockRidgeInfo{}, fmt.Errorf("invalid Rock Ridge NM length %d", length)
			}
			flags := entry[4]
			switch {
			case flags&0x02 != 0:
				info.name = "."
			case flags&0x04 != 0:
				info.name = ".."
			default:
				info.name += string(entry[5:])
			}
		case "PX":
			if length >= 12 {
				info.mode = fs.FileMode(binary.LittleEndian.Uint32(entry[4:8]))
			}
		case "SL":
			target, err := parseRockRidgeSymlink(entry)
			if err != nil {
				return rockRidgeInfo{}, err
			}
			info.symlink = target
		}
		systemUse = systemUse[length:]
	}
	return info, nil
}

func parseRockRidgeSymlink(entry []byte) (string, error) {
	if len(entry) < 5 {
		return "", fmt.Errorf("invalid Rock Ridge SL length %d", len(entry))
	}
	var parts []string
	for offset := 5; offset < len(entry); {
		if offset+2 > len(entry) {
			return "", fmt.Errorf("invalid Rock Ridge SL component at offset %d", offset)
		}
		flags := entry[offset]
		length := int(entry[offset+1])
		offset += 2
		if offset+length > len(entry) {
			return "", fmt.Errorf("invalid Rock Ridge SL component length %d", length)
		}
		switch {
		case flags&0x02 != 0:
			parts = append(parts, ".")
		case flags&0x04 != 0:
			parts = append(parts, "..")
		case flags&0x08 != 0:
			parts = append(parts, "")
		default:
			parts = append(parts, string(entry[offset:offset+length]))
		}
		if flags&0x01 != 0 {
			return "", errors.New("continued Rock Ridge symlink is not supported")
		}
		offset += length
	}
	return strings.Join(parts, "/"), nil
}

func normalizeISO9660Name(name string) string {
	if name == "\x00" {
		return "."
	}
	if name == "\x01" {
		return ".."
	}
	name = strings.TrimSuffix(name, ";1")
	name = strings.TrimRight(name, ".")
	return strings.ToLower(name)
}

func safeISOPathComponent(name string) (string, error) {
	if name == "" || name == "." || name == ".." {
		return "", fmt.Errorf("unsafe ISO path component %q", name)
	}
	if filepath.IsAbs(name) || strings.ContainsRune(name, 0) ||
		strings.Contains(name, "/") || strings.ContainsRune(name, os.PathSeparator) {
		return "", fmt.Errorf("unsafe ISO path component %q", name)
	}
	return name, nil
}
