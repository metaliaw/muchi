package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"AGENTS.md":                                 "[Rules](RULES.md)\n",
		"RULES.md":                                  "[Rule](rules/one.md)\n",
		"VALUES.md":                                 "[Value](values/one.md)\n",
		"PATTERNS.md":                               "[Pattern](patterns/one.md)\n",
		"rules/one.md":                              "[Example](../examples/demo.md)\n",
		"values/one.md":                             "Value\n",
		"patterns/one.md":                           "Pattern\n",
		".canonignore":                              "jokes/**\n**/bin/**\n",
		".gitignore":                                ".agents/settings.local.json\n",
		".agents/skills/one-two-update/SKILL.md":    "[Reload](../one-two-reload/SKILL.md)\n",
		".agents/skills/one-two-reload/SKILL.md":    "[Update](../one-two-update/SKILL.md)\n",
		".agents/skills/reader/SKILL.md":            "[Helper](../helper/SKILL.md)\n[Rule](../../../rules/one.md)\n",
		".agents/skills/helper/SKILL.md":            "[Reader](../reader/SKILL.md)\n[Notes](references/notes.md)\n",
		".agents/skills/helper/references/notes.md": "Supporting notes\n",
		".agents/skills/unused/SKILL.md":            "Not selected\n",
		".agents/skills/reader/bin/compiled":        "Do not carry\n",
		".agents/agents/dove.md":                    "[Reader](../skills/reader/SKILL.md)\n[Values](../../VALUES.md)\n",
		".agents/agents/dove.toml":                  "name = 'dove'\n",
		"examples/demo.md":                          "An optional example\n",
		"jokes/private.md":                          "Excluded source\n",
	}
	for name, content := range files {
		writeFixture(t, root, name, content)
	}
	for _, args := range [][]string{
		{"init", "-b", "main"}, {"remote", "add", "origin", sourceAddress}, {"add", "."},
		{"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-m", "Fixture"},
	} {
		if _, err := runGit(root, args...); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func writeFixture(t *testing.T, root, name, content string) {
	t.Helper()
	file := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func extractFixture(t *testing.T, archivePath string) (string, map[string][]byte) {
	t.Helper()
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	root := t.TempDir()
	files := map[string][]byte{}
	for _, file := range archive.File {
		if !strings.HasPrefix(file.Name, ".agents/") || strings.Contains(file.Name, "..") || !file.Mode().IsRegular() {
			t.Fatalf("unsafe entry: %s", file.Name)
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		files[file.Name] = content
		writeFixture(t, root, file.Name, string(content))
	}
	return root, files
}

func TestExportWorksWithoutGitOrSymlinks(t *testing.T) {
	source := createFixture(t)
	output := filepath.Join(t.TempDir(), "bundle.zip")
	if err := exportBundle([]string{"--source", source, "--output", output}, io.Discard); err != nil {
		t.Fatal(err)
	}
	root, files := extractFixture(t, output)
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != ".agents" {
		t.Fatalf("unexpected roots: %v, %v", entries, err)
	}
	for _, name := range []string{
		".agents/agents/dove.md", ".agents/agents/dove.toml", ".agents/skills/reader/SKILL.md",
		".agents/skills/helper/references/notes.md", ".agents/canon/PATTERNS.md",
	} {
		if _, ok := files[name]; !ok {
			t.Errorf("missing dependency: %s", name)
		}
	}
	for name := range files {
		if strings.Contains(name, "/unused/") || strings.Contains(name, "/bin/") || strings.Contains(name, "/jokes/") || strings.Contains(name, "/examples/") || strings.Contains(name, "/.git/") {
			t.Errorf("unrequested file: %s", name)
		}
	}
	if got := string(files[".agents/skills/reader/SKILL.md"]); !strings.Contains(got, "[Rule](../../canon/rules/one.md)") {
		t.Errorf("unrelocated rule: %s", got)
	}
	if got := string(files[".agents/agents/dove.md"]); !strings.Contains(got, "[Values](../canon/VALUES.md)") {
		t.Errorf("unrelocated values: %s", got)
	}
	for name, content := range files {
		if filepath.Ext(name) != ".md" {
			continue
		}
		for _, match := range markdownLink.FindAllSubmatch(content, -1) {
			link := string(match[1])
			if strings.Contains(link, ":") || strings.HasPrefix(link, "#") {
				continue
			}
			if _, err := os.Stat(filepath.Join(root, filepath.Dir(name), strings.Split(link, "#")[0])); err != nil {
				t.Errorf("broken extracted link %s: %v", name, err)
			}
		}
	}
	var manifest bundleManifest
	if err := json.Unmarshal(files[".agents/distribution.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if !manifest.Snapshot || manifest.Dirty || len(manifest.Commit) != 40 || len(manifest.Files) != len(files)-1 {
		t.Fatalf("wrong provenance: %+v", manifest)
	}
	if len(manifest.RemoteReferences) != 1 || manifest.RemoteReferences[0] != "examples/demo.md" {
		t.Errorf("wrong remote references: %v", manifest.RemoteReferences)
	}
	for name, digest := range manifest.Files {
		hash := sha256.Sum256(files[name])
		if digest != hex.EncodeToString(hash[:]) {
			t.Errorf("wrong checksum: %s", name)
		}
	}
	second := filepath.Join(t.TempDir(), "same.zip")
	if err := exportBundle([]string{"--source", source, "--output", second}, io.Discard); err != nil {
		t.Fatal(err)
	}
	firstBytes, _ := os.ReadFile(output)
	secondBytes, _ := os.ReadFile(second)
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Error("same source produced different archives")
	}
}

func TestExportRefusesIncompleteOrUnsafeSources(t *testing.T) {
	for _, scenario := range []string{"unknown", "dirty", "missing", "excluded", "symlink", "escape"} {
		t.Run(scenario, func(t *testing.T) {
			source := createFixture(t)
			output := filepath.Join(t.TempDir(), "bundle.zip")
			args := []string{"--source", source, "--output", output, "--allow-dirty"}
			switch scenario {
			case "unknown":
				args = append(args, "--skills", "absent")
			case "dirty":
				writeFixture(t, source, "rules/one.md", "Changed\n")
				args = args[:len(args)-1]
			case "missing":
				writeFixture(t, source, ".agents/skills/reader/SKILL.md", "[Missing](references/absent.md)\n")
			case "excluded":
				args = append(args, "--skills", "one-two-joke")
			case "escape":
				args = append(args, "--skills", "../../outside")
			case "symlink":
				file := filepath.Join(source, ".agents/skills/helper/references/notes.md")
				if err := os.Remove(file); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(source, "RULES.md"), file); err != nil {
					t.Fatal(err)
				}
			}
			if err := exportBundle(args, io.Discard); err == nil {
				t.Fatal("unsafe export succeeded")
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Errorf("failed export left output: %v", err)
			}
		})
	}
}

func TestPreviewIsLabelledAndExistingArchiveSurvives(t *testing.T) {
	source := createFixture(t)
	writeFixture(t, source, "rules/one.md", "Local rule\n")
	output := filepath.Join(t.TempDir(), "bundle.zip")
	args := []string{"--source", source, "--output", output, "--allow-dirty", "--agents", "", "--skills", "reader"}
	if err := exportBundle(args, io.Discard); err != nil {
		t.Fatal(err)
	}
	_, files := extractFixture(t, output)
	if !bytes.Contains(files[".agents/distribution.json"], []byte(`"dirty": true`)) {
		t.Error("preview was not labelled")
	}
	if _, exists := files[".agents/agents/dove.md"]; exists {
		t.Error("unselected agent was included")
	}
	before, _ := os.ReadFile(output)
	if err := exportBundle(args, io.Discard); err == nil {
		t.Fatal("existing archive overwritten")
	}
	after, _ := os.ReadFile(output)
	if !bytes.Equal(before, after) {
		t.Error("existing archive changed")
	}
}

func TestCodeExamplesSurviveRelocation(t *testing.T) {
	source := createFixture(t)
	code := "```go\nReadJSON[Body](req)\n```\nUse `ReadJSON[Body](req)` here.\n"
	writeFixture(t, source, "rules/one.md", code+"[Value](../values/one.md)\n")
	output := filepath.Join(t.TempDir(), "bundle.zip")
	if err := exportBundle([]string{"--source", source, "--output", output, "--allow-dirty"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	_, files := extractFixture(t, output)
	if got := string(files[".agents/canon/rules/one.md"]); got != code+"[Value](../values/one.md)\n" {
		t.Errorf("code changed: %s", got)
	}
}

func TestRequiredExcludedSupportFailsExport(t *testing.T) {
	source := createFixture(t)
	writeFixture(t, source, ".canonignore", "jokes/**\n**/bin/**\n.agents/skills/helper/references/notes.md\n")
	output := filepath.Join(t.TempDir(), "bundle.zip")
	if err := exportBundle([]string{"--source", source, "--output", output, "--allow-dirty"}, io.Discard); err == nil {
		t.Fatal("required support silently became a remote link")
	}
}
