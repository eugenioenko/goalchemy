package rt

import "hash/crc32"

func LibChecksumCRC32IEEE(data []byte) uint32 { return crc32.ChecksumIEEE(data) }
