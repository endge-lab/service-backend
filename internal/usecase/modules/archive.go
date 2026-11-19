package modules

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

const MaxArchiveSize = 15 << 20
const maxExpandedSize = 32 << 20
const maxManifestSize = 16 << 10

var identityPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

func validVersion(version string) bool {
	if len(version) > 128 || !versionPattern.MatchString(version) {
		return false
	}
	withoutBuild, _, _ := strings.Cut(version, "+")
	_, prerelease, _ := strings.Cut(withoutBuild, "-")
	for _, part := range strings.Split(prerelease, ".") {
		if len(part) > 1 && part[0] == '0' && strings.Trim(part, "0123456789") == "" {
			return false
		}
	}
	return true
}

func openArchive(data []byte) (map[string]*zip.File, error) {
	if len(data) == 0 || len(data) > MaxArchiveSize {
		return nil, errors.New("invalid ZIP size")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid ZIP: %w", err)
	}
	files := make(map[string]*zip.File, 3)
	var total uint64
	for _, file := range reader.File {
		if !file.Mode().IsRegular() || (file.Name != "manifest.json" && file.Name != "module.js" && file.Name != "module.css") {
			return nil, fmt.Errorf("invalid ZIP entry %q", file.Name)
		}
		if files[file.Name] != nil || file.UncompressedSize64 > maxExpandedSize-total {
			return nil, fmt.Errorf("duplicate entry or expanded size exceeded: %q", file.Name)
		}
		total += file.UncompressedSize64
		files[file.Name] = file
	}
	if files["manifest.json"] == nil || files["module.js"] == nil || files["module.js"].UncompressedSize64 == 0 {
		return nil, errors.New("manifest.json and non-empty module.js are required")
	}
	if files["manifest.json"].UncompressedSize64 > maxManifestSize {
		return nil, errors.New("manifest.json is too large")
	}
	return files, nil
}

func parseArchive(data []byte, identity, actor string) (entities.ModuleVersion, error) {
	files, err := openArchive(data)
	if err != nil {
		return entities.ModuleVersion{}, err
	}
	var manifest entities.ModuleManifest
	for name, file := range files {
		reader, err := file.Open()
		if err != nil {
			return entities.ModuleVersion{}, err
		}
		if name == "manifest.json" {
			decoder := json.NewDecoder(io.LimitReader(reader, maxManifestSize+1))
			decoder.DisallowUnknownFields()
			err = decoder.Decode(&manifest)
			if err == nil && decoder.Decode(new(any)) != io.EOF {
				err = errors.New("invalid manifest JSON")
			}
		} else {
			var size int64
			size, err = io.Copy(io.Discard, io.LimitReader(reader, int64(file.UncompressedSize64)+1))
			if err == nil && uint64(size) != file.UncompressedSize64 {
				err = errors.New("invalid unpacked size")
			}
		}
		reader.Close()
		if err != nil {
			return entities.ModuleVersion{}, fmt.Errorf("%s: %w", name, err)
		}
	}
	if manifest.Identity != identity || !validVersion(manifest.Version) || manifest.FormatVersion != 1 || manifest.HostAPI != 1 {
		return entities.ModuleVersion{}, errors.New("manifest requires the matching identity, exact SemVer, formatVersion 1 and hostApi 1")
	}
	sum := sha256.Sum256(data)
	return entities.ModuleVersion{Version: manifest.Version, Manifest: manifest, SHA256: hex.EncodeToString(sum[:]), Size: len(data), HasCSS: files["module.css"] != nil, Archive: data, PublishedBy: actor}, nil
}
