package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	emptyHash = "AAAAAAAAAAAAAAAAAAAAAAAAAAA="
	jHash     = "SgAAAAAAAAAAAAAAAQAAAAAAAAA=" // hash of "J" (0x4a)
)

func runCLI(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = run(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSingleFilePrintsBareHash(t *testing.T) {
	p := writeFile(t, t.TempDir(), "j.txt", "J")
	code, out, errOut := runCLI(t, "", p)
	if code != 0 || out != jHash+"\n" || errOut != "" {
		t.Errorf("got code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a", "J")
	b := writeFile(t, dir, "b", "")
	code, out, _ := runCLI(t, "", a, b)
	want := jHash + "  " + a + "\n" + emptyHash + "  " + b + "\n"
	if code != 0 || out != want {
		t.Errorf("got code=%d out=%q, want %q", code, out, want)
	}
}

func TestStdin(t *testing.T) {
	for _, args := range [][]string{nil, {"-"}} {
		code, out, _ := runCLI(t, "J", args...)
		if code != 0 || out != jHash+"\n" {
			t.Errorf("args %v: got code=%d out=%q", args, code, out)
		}
	}
}

func TestMissingFile(t *testing.T) {
	dir := t.TempDir()
	good := writeFile(t, dir, "good", "J")
	missing := filepath.Join(dir, "missing")
	code, out, errOut := runCLI(t, "", good, missing)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if out != jHash+"  "+good+"\n" {
		t.Errorf("stdout = %q; other files should still be hashed", out)
	}
	if !strings.Contains(errOut, missing) {
		t.Errorf("stderr = %q, want it to name %s", errOut, missing)
	}
}

func TestDirectoryIsError(t *testing.T) {
	code, out, errOut := runCLI(t, "", t.TempDir())
	if code != 1 || out != "" || !strings.Contains(errOut, "directory") {
		t.Errorf("got code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestVersion(t *testing.T) {
	code, out, _ := runCLI(t, "", "-version")
	if code != 0 || !strings.HasPrefix(out, "quickxorhash ") {
		t.Errorf("got code=%d out=%q", code, out)
	}
}

func TestUnknownFlag(t *testing.T) {
	code, out, errOut := runCLI(t, "", "-nope")
	if code != 2 || out != "" || errOut == "" {
		t.Errorf("got code=%d out=%q err=%q", code, out, errOut)
	}
}
