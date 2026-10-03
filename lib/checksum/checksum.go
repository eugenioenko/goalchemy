// Package checksum provides native, non-cryptographic checksums.
package checksum

import "hash/crc32"

// CRC32IEEE returns the IEEE CRC-32 checksum of data, as used by ZIP.
// Nil and empty slices return zero. The input is not modified.
func CRC32IEEE(data []byte) uint32 { return crc32.ChecksumIEEE(data) }
