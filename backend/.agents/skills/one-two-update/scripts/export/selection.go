package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var markdownLink = regexp.MustCompile(`\]\(([^\s)]+)\)`)
var markdownToken = regexp.MustCompile("`+[^`]*`+|" + markdownLink.String())
var selectionName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type bundleManifest struct {
	Schema           int               `json:"schema"`
	Source           string            `json:"source"`
	Commit           string            `json:"commit"`
	Dirty            bool              `json:"dirty"`
	Snapshot         bool              `json:"snapshot"`
	Skills           []string          `json:"skills"`
	Agents           []string          `json:"agents"`
	RemoteReferences []string          `json:"remote_references,omitempty"`
	Files            map[string]string `json:"files"`
}

type bundleSelection struct {
	source sourceTree
	files  map[string][]byte
	skills map[string]bool
	agents map[string]bool
}

func collectBundle(source sourceTree, options exportOptions) (map[string][]byte, error) {
	selection := bundleSelection{source, map[string][]byte{}, map[string]bool{}, map[string]bool{}}
	if err := selection.addCanon(); err != nil {
		return nil, err
	}
	for _, skill := range strings.Split("one-two-update,one-two-reload,"+options.skills, ",") {
		if skill != "" {
			if err := selection.addSkill(skill); err != nil {
				return nil, err
			}
		}
	}
	for _, agent := range strings.Split(options.agents, ",") {
		if agent != "" {
			if err := selection.addAgent(agent); err != nil {
				return nil, err
			}
		}
	}
	return selection.renderBundle()
}

func (selection *bundleSelection) addCanon() error {
	for _, name := range []string{"AGENTS.md", "RULES.md", "VALUES.md", "PATTERNS.md", ".canonignore"} {
		if err := selection.addFile(name); err != nil {
			return err
		}
	}
	for _, prefix := range []string{"rules/", "values/", "patterns/"} {
		if err := selection.addDirectory(prefix); err != nil {
			return err
		}
	}
	return nil
}

func (selection *bundleSelection) addDirectory(prefix string) error {
	for name := range selection.source.files {
		if strings.HasPrefix(name, prefix) && !selection.source.excluded[name] {
			if err := selection.addFile(name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (selection *bundleSelection) addSkill(name string) error {
	if !selectionName.MatchString(name) {
		return fmt.Errorf("invalid skill name: %s", name)
	}
	if name == "one-two-joke" {
		return fmt.Errorf("one-two-joke requires excluded hand-written material; it cannot be bundled")
	}
	if selection.skills[name] {
		return nil
	}
	selection.skills[name] = true
	prefix := ".agents/skills/" + name + "/"
	if err := selection.addFile(prefix + "SKILL.md"); err != nil {
		return err
	}
	return selection.addDirectory(prefix)
}

func (selection *bundleSelection) addAgent(name string) error {
	if !selectionName.MatchString(name) {
		return fmt.Errorf("invalid agent name: %s", name)
	}
	if selection.agents[name] {
		return nil
	}
	selection.agents[name] = true
	if err := selection.addFile(".agents/agents/" + name + ".md"); err != nil {
		return err
	}
	adapter := ".agents/agents/" + name + ".toml"
	if selection.source.files[adapter] {
		return selection.addFile(adapter)
	}
	return nil
}

func (selection *bundleSelection) addFile(name string) error {
	if _, exists := selection.files[name]; exists {
		return nil
	}
	content, err := selection.source.readFile(name)
	if err != nil {
		return err
	}
	selection.files[name] = content
	if path.Ext(name) != ".md" {
		return nil
	}
	_, err = visitLinks(content, func(link string) (string, error) {
		target, _, local := resolveLink(name, link)
		if !local {
			return link, nil
		}
		parts := strings.Split(target, "/")
		if len(parts) >= 3 && parts[0] == ".agents" {
			switch parts[1] {
			case "skills":
				err = selection.addSkill(parts[2])
			case "agents":
				err = selection.addAgent(strings.TrimSuffix(parts[2], path.Ext(parts[2])))
			}
		}
		return link, err
	})
	return err
}

func visitLinks(content []byte, replace func(string) (string, error)) ([]byte, error) {
	lines := strings.SplitAfter(string(content), "\n")
	fence := ""
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[0]
			length := 0
			for length < len(trimmed) && trimmed[length] == marker {
				length++
			}
			fence = trimmed[:length]
			continue
		}
		var failure error
		lines[index] = markdownToken.ReplaceAllStringFunc(line, func(token string) string {
			match := markdownLink.FindStringSubmatch(token)
			if strings.HasPrefix(token, "`") || match == nil {
				return token
			}
			link, err := replace(match[1])
			if err != nil {
				failure = err
				return token
			}
			return "](" + link + ")"
		})
		if failure != nil {
			return nil, failure
		}
	}
	return []byte(strings.Join(lines, "")), nil
}

func resolveLink(file, link string) (string, string, bool) {
	if strings.Contains(link, ":") || strings.HasPrefix(link, "#") {
		return "", "", false
	}
	name, anchor, found := strings.Cut(link, "#")
	if found {
		anchor = "#" + anchor
	}
	return path.Clean(path.Join(path.Dir(file), name)), anchor, true
}

func mapDestination(name string) string {
	if strings.HasPrefix(name, ".agents/") {
		return name
	}
	return path.Join(".agents/canon", name)
}

func containsPath[T any](files map[string]T, name string) bool {
	if _, ok := files[name]; ok {
		return true
	}
	for file := range files {
		if strings.HasPrefix(file, name+"/") {
			return true
		}
	}
	return false
}

func listNames[T any](items map[string]T) []string {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (selection *bundleSelection) renderBundle() (map[string][]byte, error) {
	bundle := make(map[string][]byte)
	manifest := bundleManifest{Schema: 1, Source: sourceAddress, Commit: selection.source.commit,
		Dirty: selection.source.dirty, Snapshot: true, Skills: listNames(selection.skills),
		Agents: listNames(selection.agents), Files: map[string]string{}}
	remote := map[string]bool{}
	for _, name := range listNames(selection.files) {
		content := selection.files[name]
		if path.Ext(name) == ".md" {
			var err error
			content, err = selection.rewriteLinks(name, content, remote)
			if err != nil {
				return nil, err
			}
		}
		destination := mapDestination(name)
		bundle[destination] = content
		digest := sha256.Sum256(content)
		manifest.Files[destination] = hex.EncodeToString(digest[:])
	}
	manifest.RemoteReferences = listNames(remote)
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	bundle[".agents/distribution.json"] = append(content, '\n')
	return bundle, nil
}

func (selection *bundleSelection) rewriteLinks(name string, content []byte, remote map[string]bool) ([]byte, error) {
	return visitLinks(content, func(link string) (string, error) {
		target, anchor, local := resolveLink(name, link)
		if !local {
			return link, nil
		}
		if containsPath(selection.files, target) {
			relative, err := filepath.Rel(filepath.FromSlash(path.Dir(mapDestination(name))), filepath.FromSlash(mapDestination(target)))
			return filepath.ToSlash(relative) + anchor, err
		}
		if strings.HasPrefix(target, ".agents/") || strings.HasPrefix(target, "rules/") || strings.HasPrefix(target, "values/") || strings.HasPrefix(target, "patterns/") || !containsPath(selection.source.files, target) {
			return "", fmt.Errorf("unresolved reference in %s: %s", name, link)
		}
		remote[target] = true
		return "https://github.com/cangrejometralleta/OneTwoThree/blob/" + selection.source.commit + "/" + target + anchor, nil
	})
}
