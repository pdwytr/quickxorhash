// Command quickxorhash prints the Microsoft Graph / OneDrive QuickXorHash of
// files, base64-encoded exactly as Graph reports file.hashes.quickXorHash.
//
// Usage:
//
//	quickxorhash [-version] [file ...]
//
// With one file it prints the bare hash. With several it prints
// "<hash>  <path>" per file. With no arguments, or "-", it reads stdin.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/pdwytr/quickxorhash"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("quickxorhash", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version and exit")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: quickxorhash [-version] [file ...]")
		fmt.Fprintln(stderr, "")
		fmt.Fprintln(stderr, "Prints the base64 QuickXorHash used by Microsoft Graph / OneDrive.")
		fmt.Fprintln(stderr, "One file: bare hash. Several files: \"<hash>  <path>\" per line.")
		fmt.Fprintln(stderr, "No file, or \"-\": read standard input.")
		fmt.Fprintln(stderr, "")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, "quickxorhash", resolveVersion())
		return 0
	}

	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"-"}
	}

	status := 0
	for _, p := range paths {
		sum, err := hashPath(p, stdin)
		if err != nil {
			fmt.Fprintf(stderr, "quickxorhash: %s: %v\n", p, err)
			status = 1
			continue
		}
		if len(paths) == 1 {
			fmt.Fprintln(stdout, sum)
		} else {
			fmt.Fprintf(stdout, "%s  %s\n", sum, p)
		}
	}
	return status
}

func hashPath(path string, stdin io.Reader) (string, error) {
	if path == "-" {
		return quickxorhash.FromReader(stdin)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", unwrapPathError(err)
	}
	defer f.Close()
	if fi, err := f.Stat(); err == nil && fi.IsDir() {
		return "", fmt.Errorf("is a directory")
	}
	return quickxorhash.FromReader(f)
}

// unwrapPathError drops the path from *os.PathError, since callers already
// prefix messages with it.
func unwrapPathError(err error) error {
	if pe, ok := err.(*os.PathError); ok {
		return pe.Err
	}
	return err
}

// resolveVersion prefers the ldflags value, then the module version recorded
// by "go install module@version".
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}
