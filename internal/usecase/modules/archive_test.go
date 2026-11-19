package modules

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

type entry struct{ name, content string }

func makeArchive(t *testing.T, files ...entry) []byte {
	t.Helper()
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	for _, file := range files {
		target, err := writer.Create(file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := target.Write([]byte(file.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestParseModuleArchive(t *testing.T) {
	manifest := entry{"manifest.json", `{"identity":"slot-ui","version":"0.1.0-dev.2+build","formatVersion":1,"hostApi":1}`}
	js := entry{"module.js", "export default function load() {}"}
	css := entry{"module.css", "td { color: orange }"}
	data := makeArchive(t, manifest, js, css)
	version, err := parseArchive(data, "slot-ui", "publisher")
	if err != nil {
		t.Fatal(err)
	}
	if version.Version != "0.1.0-dev.2+build" || version.PublishedBy != "publisher" ||
		version.Size != len(data) || len(version.SHA256) != 64 || !version.HasCSS {
		t.Fatalf("incorrect metadata: %#v", version)
	}
	asset, err := new(UseCase).ReadAsset(version.Archive, "module.js")
	if err != nil || string(asset) != js.content {
		t.Fatalf("incorrect module.js: %s, %v", asset, err)
	}

	for name, files := range map[string][]entry{
		"no js":          {manifest},
		"path traversal": {manifest, js, {"../module.css", "bad"}},
		"duplicate":      {manifest, js, css, css},
		"unknown":        {manifest, js, {"package.json", "{}"}},
		"bad version":    {{"manifest.json", strings.Replace(manifest.content, "dev.2", "dev.02", 1)}, js},
		"wrong format":   {{"manifest.json", strings.Replace(manifest.content, "\"formatVersion\":1", "\"formatVersion\":2", 1)}, js},
		"too large":      {manifest, js, {"module.css", strings.Repeat(" ", maxExpandedSize)}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseArchive(makeArchive(t, files...), "slot-ui", "publisher"); err == nil {
				t.Fatal("invalid ZIP accepted")
			}
		})
	}
	if _, err := parseArchive(data, "different-id", "publisher"); err == nil {
		t.Fatal("accepted ZIP for a different module")
	}
}

func TestParseModuleRendererContributions(t *testing.T) {
	base := `{"identity":"slot-ui","version":"1.0.0","formatVersion":1,"hostApi":1,"contributes":{"renderers":[%s]}}`
	renderer := `{"primitive":"Table","id":"slot","label":"Slot"}`
	manifest := func(value string) entry { return entry{"manifest.json", strings.Replace(base, "%s", value, 1)} }
	js := entry{"module.js", "export default function load() {}"}
	version, err := parseArchive(makeArchive(t, manifest(renderer), js), "slot-ui", "publisher")
	if err != nil {
		t.Fatal(err)
	}
	if version.Manifest.Contributes == nil || len(version.Manifest.Contributes.Renderers) != 1 || version.Manifest.Contributes.Renderers[0].ID != "slot" {
		t.Fatalf("renderer descriptor was not preserved: %#v", version.Manifest)
	}
	for _, value := range []string{renderer + "," + renderer, strings.Replace(renderer, "Table", "table", 1), strings.Replace(renderer, `"label":"Slot"`, `"label":""`, 1)} {
		if _, err := parseArchive(makeArchive(t, manifest(value), js), "slot-ui", "publisher"); err == nil {
			t.Fatalf("accepted invalid renderer descriptors %s", value)
		}
	}
}
