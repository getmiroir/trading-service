package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/getmiroir/trading-service/internal/settings/settingsdoc"
)

func main() {
	outPath := flag.String("o", "", "output markdown file path (required)")
	flag.Parse()
	if *outPath == "" {
		fmt.Fprintln(os.Stderr, "usage: settings-doc -o output_file.md")
		os.Exit(2)
	}

	var buf bytes.Buffer
	if err := settingsdoc.Generate(&buf, moduleRoot()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, buf.Bytes(), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func moduleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}
