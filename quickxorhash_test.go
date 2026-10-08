package quickxorhash

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"testing/iotest"
)

// gen returns n deterministic pseudo-random bytes (xorshift32). The same
// generator produced the expected values in generatedVectors.
func gen(n int) []byte {
	b := make([]byte, n)
	x := uint32(2463534242)
	for i := range b {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		b[i] = byte(x >> 24)
	}
	return b
}

// generatedVectors were computed with rclone v1.69.0's quickxorhash package
// over gen(size). Sizes straddle the 64-byte block size, the 160-byte
// register width and the 1760-byte cycle buffer, plus two large inputs.
var generatedVectors = []struct {
	size int
	out  string
}{
	{0, "AAAAAAAAAAAAAAAAAAAAAAAAAAA="},
	{1, "KwAAAAAAAAAAAAAAAQAAAAAAAAA="},
	{63, "tf33VU9EDr6oovVk7zEQG7nDvys="},
	{64, "tf33VU9EzquoovVkkDEQG7nDvys="},
	{65, "tf33VU9EzqtLovVkkTEQG7nDvys="},
	{159, "SUyy27AFKlGfUcTM2itTyVZ+Vrw="},
	{160, "SUyy27AFKlGfUcTM5StTyVZ+Nq0="},
	{161, "y0yy27AFKlGfUcTM5CtTyVZ+Nq0="},
	{1759, "IGqL//aFit3bDmHXFsNb6r9crT0="},
	{1760, "IGqL//aFit3bDmHXKcNb6r9czT4="},
	{1761, "sGqL//aFit3bDmHXKMNb6r9czT4="},
	{3519, "bIbQvabTNTm9iJLHVaK3uBVt2Ug="},
	{3520, "bIbQvabTNTm9iJLHKqK3uBVtOUY="},
	{3521, "RYbQvabTNTm9iJLHK6K3uBVtOUY="},
	{65536, "UWQfMqhdl6VU3FAFkmCTsTqhlis="},
	{1<<20 + 13, "U+ZrxY0IkyUlamStRqksDBAtheE="},
}

// reference is a direct, bit-at-a-time transcription of Microsoft's
// description: input bit b of byte i flips register bit (11*i + b) mod 160,
// then the length is XORed little-endian into the last 8 bytes. It shares
// no code with the optimized implementation.
func reference(data []byte) [Size]byte {
	var h [Size]byte
	for i, c := range data {
		for b := 0; b < 8; b++ {
			if c&(1<<b) == 0 {
				continue
			}
			k := (shift*i + b) % widthInBits
			h[k/8] ^= 1 << (k % 8)
		}
	}
	l := uint64(len(data))
	for i := 0; i < 8; i++ {
		h[Size-8+i] ^= byte(l >> (8 * i))
	}
	return h
}

func mustDecode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("decode %q: %v", s, err)
	}
	return b
}

func TestRcloneVectors(t *testing.T) {
	for _, v := range rcloneVectors {
		in := mustDecode(t, v.in)
		if len(in) != v.size {
			t.Fatalf("vector size %d decodes to %d bytes", v.size, len(in))
		}
		got := Sum(in)
		if s := EncodeToString(got[:]); s != v.out {
			t.Errorf("size %d: got %s, want %s", v.size, s, v.out)
		}
	}
}

func TestGeneratedVectors(t *testing.T) {
	for _, v := range generatedVectors {
		got := Sum(gen(v.size))
		if s := EncodeToString(got[:]); s != v.out {
			t.Errorf("size %d: got %s, want %s", v.size, s, v.out)
		}
	}
}

func TestEmpty(t *testing.T) {
	const want = "AAAAAAAAAAAAAAAAAAAAAAAAAAA="
	if s := EncodeToString(New().Sum(nil)); s != want {
		t.Errorf("New().Sum(nil) = %s, want %s", s, want)
	}
	h := New()
	h.Write(nil)
	h.Write([]byte{})
	if s := EncodeToString(h.Sum(nil)); s != want {
		t.Errorf("after empty writes = %s, want %s", s, want)
	}
	if s, err := FromReader(strings.NewReader("")); err != nil || s != want {
		t.Errorf("FromReader(\"\") = %s, %v; want %s", s, err, want)
	}
}

func TestMatchesReference(t *testing.T) {
	sizes := []int{1, 2, 7, 8, 19, 20, 21, 159, 160, 161, 319, 320, 1759, 1760, 1761, 3520, 5000}
	for _, n := range sizes {
		data := gen(n)
		if got, want := Sum(data), reference(data); got != want {
			t.Errorf("size %d: got %x, want %x", n, got, want)
		}
	}
	// Length bytes must carry past the low byte.
	data := gen(70000)
	if got, want := Sum(data), reference(data); got != want {
		t.Errorf("size 70000: got %x, want %x", got, want)
	}
}

// TestChunkedEqualsOneShot writes inputs in pieces of many sizes, including
// sizes that leave partial fills of the 1760-byte cycle buffer, and checks
// the result matches a single Sum call.
func TestChunkedEqualsOneShot(t *testing.T) {
	data := gen(3*cycleLen + 123)
	want := Sum(data)
	chunks := []int{1, 2, 3, 7, 11, 63, 64, 65, 159, 160, 161, 1000, cycleLen - 1, cycleLen, cycleLen + 1, 2*cycleLen + 5}
	for _, c := range chunks {
		t.Run(fmt.Sprintf("chunk=%d", c), func(t *testing.T) {
			h := New()
			for i := 0; i < len(data); i += c {
				end := min(i+c, len(data))
				n, err := h.Write(data[i:end])
				if n != end-i || err != nil {
					t.Fatalf("Write returned %d, %v; want %d, nil", n, err, end-i)
				}
			}
			if got := h.Sum(nil); !bytes.Equal(got, want[:]) {
				t.Errorf("got %x, want %x", got, want)
			}
		})
	}
}

// TestSplitAtEveryBoundary splits one input into two writes at every offset
// across the cycle-buffer boundary.
func TestSplitAtEveryBoundary(t *testing.T) {
	data := gen(cycleLen + 200)
	want := Sum(data)
	for split := cycleLen - 100; split <= len(data); split++ {
		h := New()
		h.Write(data[:split])
		h.Write(data[split:])
		if got := h.Sum(nil); !bytes.Equal(got, want[:]) {
			t.Fatalf("split %d: got %x, want %x", split, got, want)
		}
	}
}

func TestFromReaderSmallReads(t *testing.T) {
	data := gen(5000)
	want := Sum(data)
	got, err := FromReader(iotest.OneByteReader(bytes.NewReader(data)))
	if err != nil {
		t.Fatal(err)
	}
	if got != EncodeToString(want[:]) {
		t.Errorf("got %s, want %s", got, EncodeToString(want[:]))
	}
}

func TestFromReaderError(t *testing.T) {
	if _, err := FromReader(iotest.ErrReader(iotest.ErrTimeout)); err != iotest.ErrTimeout {
		t.Errorf("err = %v, want %v", err, iotest.ErrTimeout)
	}
}

func TestSumDoesNotMutate(t *testing.T) {
	h := New()
	h.Write(gen(100))
	a := h.Sum(nil)
	b := h.Sum(nil)
	if !bytes.Equal(a, b) {
		t.Fatalf("repeated Sum differs: %x vs %x", a, b)
	}
	h.Write(gen(50))
	data := append(gen(100), gen(50)...)
	want := Sum(data)
	if got := h.Sum(nil); !bytes.Equal(got, want[:]) {
		t.Errorf("Write after Sum: got %x, want %x", got, want)
	}
}

func TestSumAppends(t *testing.T) {
	prefix := []byte("prefix")
	h := New()
	h.Write([]byte("abc"))
	out := h.Sum(append([]byte(nil), prefix...))
	want := Sum([]byte("abc"))
	if !bytes.Equal(out[:len(prefix)], prefix) || !bytes.Equal(out[len(prefix):], want[:]) {
		t.Errorf("Sum(prefix) = %x", out)
	}
}

func TestReset(t *testing.T) {
	h := New()
	empty := h.Sum(nil)
	h.Write([]byte{1})
	if bytes.Equal(empty, h.Sum(nil)) {
		t.Fatal("hash did not change after Write")
	}
	h.Reset()
	if got := h.Sum(nil); !bytes.Equal(got, empty) {
		t.Errorf("after Reset: got %x, want %x", got, empty)
	}
}

func TestSizes(t *testing.T) {
	h := New()
	if h.Size() != 20 {
		t.Errorf("Size() = %d, want 20", h.Size())
	}
	if h.BlockSize() != 64 {
		t.Errorf("BlockSize() = %d, want 64", h.BlockSize())
	}
}

func BenchmarkWrite1MiB(b *testing.B) {
	buf := gen(1 << 20)
	h := New()
	b.SetBytes(int64(len(buf)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Reset()
		h.Write(buf)
		h.Sum(nil)
	}
}
