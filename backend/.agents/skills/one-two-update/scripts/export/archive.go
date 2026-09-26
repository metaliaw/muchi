package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func validateBundle(bundle map[string][]byte) error {
	for name, content := range bundle {
		if path.Clean(name) != name || !strings.HasPrefix(name, ".agents/") || strings.Contains(name, "/.git/") {
			return fmt.Errorf("invalid archive path: %s", name)
		}
		if path.Ext(name) != ".md" {
			continue
		}
		_, err := visitLinks(content, func(link string) (string, error) {
			target, _, local := resolveLink(name, link)
			if local && !containsPath(bundle, target) {
				return "", fmt.Errorf("broken bundled link: %s -> %s", name, target)
			}
			return link, nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func writeBundle(output string, bundle map[string][]byte) error {
	if err := validateBundle(bundle); err != nil {
		return err
	}
	staged, err := os.CreateTemp(filepath.Dir(output), ".onetwothree-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(staged.Name())
	defer staged.Close()
	if err := writeArchive(staged, bundle); err != nil {
		return err
	}
	if err := staged.Close(); err != nil {
		return err
	}
	if err := verifyArchive(staged.Name(), bundle); err != nil {
		return err
	}

	destination, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	input, err := os.Open(staged.Name())
	if err != nil {
		destination.Close()
		os.Remove(output)
		return err
	}
	defer input.Close()
	_, copyErr := io.Copy(destination, input)
	closeErr := destination.Close()
	if copyErr != nil {
		os.Remove(output)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(output)
		return closeErr
	}
	return nil
}

func writeArchive(output io.Writer, bundle map[string][]byte) error {
	writer := zip.NewWriter(output)
	for _, name := range listNames(bundle) {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0644)
		if strings.HasSuffix(name, ".sh") {
			header.SetMode(0755)
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			writer.Close()
			return err
		}
		if _, err := entry.Write(bundle[name]); err != nil {
			writer.Close()
			return err
		}
	}
	return writer.Close()
}

func verifyArchive(name string, bundle map[string][]byte) error {
	archive, err := zip.OpenReader(name)
	if err != nil {
		return err
	}
	defer archive.Close()
	if len(archive.File) != len(bundle) {
		return fmt.Errorf("archive file count differs")
	}
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			return err
		}
		content, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			return err
		}
		if !file.Mode().IsRegular() || !bytes.Equal(content, bundle[file.Name]) {
			return fmt.Errorf("archive verification failed: %s", file.Name)
		}
	}
	return nil
}
