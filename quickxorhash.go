// Package quickxorhash implements QuickXorHash, the 160-bit non-cryptographic
// checksum that Microsoft Graph and OneDrive report as file.hashes.quickXorHash.
//
// The algorithm XORs each input byte into a 160-bit register at a bit offset
// that advances by 11 bits per byte (wrapping at 160), then XORs the total
// input length, as a little-endian 64-bit integer, into the last 8 bytes of
// the result. Graph returns the 20-byte result base64-encoded; see
// EncodeToString.
//
// Specification:
// https://learn.microsoft.com/onedrive/developer/code-snippets/quickxorhash
//
// The buffered approach used here (fold input into a 1760-byte cycle buffer,
// rotate into the 160-bit register only when Sum is called) follows the
// implementation in rclone's backend/onedrive/quickxorhash package
// (https://github.com/rclone/rclone, MIT License, Copyright (c) 2012 Nick
// Craig-Wood), which in turn ports namazso's QuickXorHash
// (https://github.com/namazso/QuickXorHash, BSD Zero Clause License).
package quickxorhash

import (
	"crypto/subtle"
	"encoding/base64"
	"hash"
	"io"
)

const (
	// Size is the size of a QuickXorHash checksum in bytes.
	Size = 20

	// BlockSize is the block size reported by hash.Hash. Write accepts any
	// length; this only advises callers on an efficient write size.
	BlockSize = 64

	widthInBits = 8 * Size // 160-bit register
	shift       = 11       // bit offset advance per input byte

	// After widthInBits bytes the bit offset returns to where it started
	// (11*160 is a multiple of 160), so byte i and byte i+cycleLen land on
	// the same bit position. Bytes that collide can be XORed together first
	// and rotated into the register once, at Sum time.
	cycleLen = shift * widthInBits // 1760 bytes
)

// digest is the running QuickXorHash state.
type digest struct {
	cycle  [cycleLen]byte // input bytes XOR-folded modulo cycleLen
	length uint64         // total bytes written
}

// Compile-time check that *digest satisfies hash.Hash.
var _ hash.Hash = (*digest)(nil)

// New returns a new hash.Hash computing the QuickXorHash checksum.
func New() hash.Hash {
	return new(digest)
}

// Write adds p to the running hash. It always returns len(p), nil.
func (d *digest) Write(p []byte) (int, error) {
	n := len(p)
	pos := int(d.length % cycleLen)
	for len(p) > 0 {
		// XORBytes processes min(len(dst), len(src)) bytes.
		k := subtle.XORBytes(d.cycle[pos:], d.cycle[pos:], p)
		p = p[k:]
		pos = 0
	}
	d.length += uint64(n)
	return n, nil
}

// Sum appends the current checksum to b and returns the resulting slice.
// It does not change the underlying hash state.
func (d *digest) Sum(b []byte) []byte {
	h := d.checksum()
	return append(b, h[:]...)
}

// checksum folds the cycle buffer into the 160-bit register and mixes in
// the length.
func (d *digest) checksum() [Size]byte {
	// One spare byte absorbs the carry from bit offsets 153..159, whose
	// high bits wrap around to the start of the register.
	var h [Size + 1]byte
	for i, c := range d.cycle {
		bit := (i * shift) % widthInBits
		v := uint16(c) << (bit % 8)
		h[bit/8] ^= byte(v)
		h[bit/8+1] ^= byte(v >> 8)
	}
	h[0] ^= h[Size]

	l := d.length
	for i := Size - 8; i < Size; i++ {
		h[i] ^= byte(l)
		l >>= 8
	}

	var out [Size]byte
	copy(out[:], h[:Size])
	return out
}

// Reset resets the hash to its initial state.
func (d *digest) Reset() { *d = digest{} }

// Size returns the number of bytes Sum appends: 20.
func (d *digest) Size() int { return Size }

// BlockSize returns the hash's preferred write size.
func (d *digest) BlockSize() int { return BlockSize }

// Sum returns the QuickXorHash checksum of data.
func Sum(data []byte) [Size]byte {
	var d digest
	_, _ = d.Write(data)
	return d.checksum()
}

// EncodeToString returns the standard base64 encoding of a checksum, the
// form Microsoft Graph uses for quickXorHash.
func EncodeToString(sum []byte) string {
	return base64.StdEncoding.EncodeToString(sum)
}

// FromReader hashes everything read from r and returns the base64-encoded
// checksum, as Microsoft Graph reports it.
func FromReader(r io.Reader) (string, error) {
	h := New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return EncodeToString(h.Sum(nil)), nil
}
