package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type exportOptions struct {
	source     string
	output     string
	skills     string
	agents     string
	allowDirty bool
}

func exportBundle(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("export", flag.ContinueOnError)
	flags.SetOutput(out)
	var options exportOptions
	flags.StringVar(&options.source, "source", "", "Source repository root")
	flags.StringVar(&options.output, "output", "", "New ZIP file path")
	flags.StringVar(&options.skills, "skills", "", "Comma-separated skill names")
	flags.StringVar(&options.agents, "agents", "dove", "Comma-separated agents; empty selects none")
	flags.BoolVar(&options.allowDirty, "allow-dirty", false, "Export a labelled working-tree snapshot")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !filepath.IsAbs(options.source) || !filepath.IsAbs(options.output) || flags.NArg() != 0 {
		return fmt.Errorf("use absolute paths: --source <repository> --output <new.zip> [--skills <names>]")
	}

	source, err := readSourceTree(options)
	if err != nil {
		return err
	}
	bundle, err := collectBundle(source, options)
	if err != nil {
		return err
	}
	if err := writeBundle(options.output, bundle); err != nil {
		return err
	}
	fmt.Fprintf(out, "✅ Snapshot Exported: %s (%d files)\n", options.output, len(bundle))
	return nil
}

func main() {
	if err := exportBundle(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "❌ Export Failed:", err)
		os.Exit(1)
	}
}
