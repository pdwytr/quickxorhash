# quickxorhash

Compute the Microsoft Graph / OneDrive **QuickXorHash** of local files, base64-encoded exactly as Graph reports `file.hashes.quickXorHash`.

Graph returns QuickXorHash for files in OneDrive and SharePoint, but nothing that ships with Windows or macOS can compute it (`Get-FileHash` and `shasum` don't support it). Without a local hash you can't confirm that an upload or migration arrived intact. This tool fills that gap: a single static binary with no dependencies, plus a Go package that implements `hash.Hash`.

It has been used in production migration workflows to verify files before and after upload.

## Install

### Prebuilt binaries

Download the archive for your platform from the [releases page](https://github.com/pdwytr/quickxorhash/releases). Builds are published for macOS, Linux and Windows on amd64 and arm64. Each release includes `checksums.txt`:

```bash
sha256sum --ignore-missing -c checksums.txt        # Linux
shasum -a 256 --ignore-missing -c checksums.txt    # macOS
```

```powershell
Get-FileHash .\quickxorhash_v1.0.0_windows_amd64.zip -Algorithm SHA256   # compare with checksums.txt
```

### With Go (1.22+)

```bash
go install github.com/pdwytr/quickxorhash/cmd/quickxorhash@latest
```

## Usage

```text
quickxorhash [-version] [file ...]
```

- One file: prints the bare hash.
- Several files: prints `<hash>  <path>` per line.
- No arguments or `-`: reads standard input.
- Errors go to stderr. The exit status is 1 if any file failed and 2 for bad flags. Files that can be read are still hashed.

macOS / Linux:

```bash
$ printf 'hello world\n' > hello.txt
$ quickxorhash hello.txt
aCgDG9jwBhDc4Q1ybAMZFAAAAAA=

$ quickxorhash hello.txt notes.txt
aCgDG9jwBhDc4Q1ybAMZFAAAAAA=  hello.txt
MKYwSORAhzLIgQ15EoEb6tAGMZQ=  notes.txt

$ quickxorhash < hello.txt
aCgDG9jwBhDc4Q1ybAMZFAAAAAA=
```

Windows (PowerShell):

```powershell
PS> .\quickxorhash.exe "C:\Data\hello.txt"
aCgDG9jwBhDc4Q1ybAMZFAAAAAA=

PS> Get-ChildItem C:\Data -File -Recurse | ForEach-Object { .\quickxorhash.exe $_.FullName }
```

PowerShell 5.1 can re-encode bytes it pipes to a native command, so pass file paths as arguments rather than piping file contents.

## Compare against Microsoft Graph

Ask Graph for the item's `file` facet:

```http
GET https://graph.microsoft.com/v1.0/me/drive/items/{item-id}?$select=name,size,file
```

```json
{
  "name": "hello.txt",
  "size": 12,
  "file": {
    "mimeType": "text/plain",
    "hashes": { "quickXorHash": "aCgDG9jwBhDc4Q1ybAMZFAAAAAA=" }
  }
}
```

The file is intact if `file.hashes.quickXorHash` matches this tool's output byte-for-byte. Use `/drives/{drive-id}/items/{item-id}` or `/sites/{site-id}/drive/items/{item-id}` for other drives.

With the Microsoft Graph PowerShell SDK:

```powershell
$item  = Invoke-MgGraphRequest -Uri "https://graph.microsoft.com/v1.0/me/drive/items/${itemId}?`$select=file"
$local = .\quickxorhash.exe "C:\Data\hello.txt"
$item.file.hashes.quickXorHash -ceq $local
```

The comparison is case-sensitive because base64 is case-sensitive.

## Library

```go
import "github.com/pdwytr/quickxorhash"

// Streaming: quickxorhash.New() returns a hash.Hash.
h := quickxorhash.New()
io.Copy(h, f)
fmt.Println(quickxorhash.EncodeToString(h.Sum(nil)))

// From an io.Reader, base64 result.
s, err := quickxorhash.FromReader(f)

// One-shot over a byte slice: returns [20]byte.
sum := quickxorhash.Sum(data)
```

The package uses only the standard library.

## Algorithm

QuickXorHash XORs each input byte into a 160-bit register at a bit offset that advances 11 bits per byte, wrapping at 160. It then XORs the input length, as a little-endian 64-bit integer, into the last 8 bytes. It detects transfer corruption and is **not** a cryptographic hash. See Microsoft's [reference description](https://learn.microsoft.com/onedrive/developer/code-snippets/quickxorhash).

The implementation is tested against rclone's known-answer vectors, against vectors generated with rclone's implementation, and against a bit-by-bit reference implementation in the test suite.

## License

[MIT](LICENSE). Test vectors and the buffered design follow [rclone](https://github.com/rclone/rclone)'s `quickxorhash` package (MIT).
