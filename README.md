# quickxorhash

A small CLI utility that computes **QuickXorHash** for a file and prints the result in **base64**.

This is useful when you need hash parity with Microsoft Graph API / OneDrive metadata.  
Windows does not provide a built-in QuickXorHash command, so this project provides a reliable way to generate it locally.

The Go implementation uses the proven `rclone` QuickXorHash package and has been used in production migration workflows.

## Why this exists

- Microsoft Graph and OneDrive expose `quickXorHash` for files.
- Built-in Windows hashing tools (for example `Get-FileHash`) do not support QuickXorHash.
- Teams often need local hash verification before/after upload or during migration.

## Output format

The tool prints a single base64-encoded QuickXorHash value:

```text
AAAAAAAAAAAAAAAAAAAAAAAAAAA=
```

## Build

### Build for current platform

```bash
go build -o quickxorhash quickxorhash.go
```

### Build for Windows from any platform

```bash
GOOS=windows GOARCH=amd64 go build -o quickxorhash.exe quickxorhash.go
```

PowerShell equivalent:

```powershell
$env:GOOS="windows"
$env:GOARCH="amd64"
go build -o quickxorhash.exe quickxorhash.go
```

## Usage

### Single file hash

macOS/Linux:

```bash
./quickxorhash "/path/to/file.ext"
```

Windows (PowerShell):

```powershell
.\quickxorhash.exe "C:\path\to\file.ext"
```
