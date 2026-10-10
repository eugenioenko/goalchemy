// SPDX-License-Identifier: Apache-2.0

// Package os reads and writes whole host files. Results carry a status code
// instead of an error so every host reports the same failure classes;
// std/os turns them into *PathError values.
package os

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"syscall"
)

// MaxFileBytes is the largest file ReadFile returns and WriteFile accepts.
const MaxFileBytes = 1 << 30

// Status codes. Hosts without a file system report Unsupported.
const (
	OK          = 0
	NotExist    = 1
	Exist       = 2
	Permission  = 3
	IsDir       = 4
	NotDir      = 5
	TooLarge    = 6
	Unsupported = 7
	Invalid     = 8
	IO          = 9
)

// ReadFile returns the contents of the named file.
func ReadFile(name string) ([]byte, int) {
	if strings.IndexByte(name, 0) >= 0 {
		return nil, Invalid
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, status(err)
	}
	defer f.Close()
	if st, err := f.Stat(); err == nil && st.Size() > MaxFileBytes {
		return nil, TooLarge
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
	if err != nil {
		return nil, status(err)
	}
	if len(data) > MaxFileBytes {
		return nil, TooLarge
	}
	return data, OK
}

// WriteFile writes data to the named file, creating it with perm (before the
// process umask) if necessary and truncating it otherwise.
func WriteFile(name string, data []byte, perm uint32) int {
	if strings.IndexByte(name, 0) >= 0 {
		return Invalid
	}
	if len(data) > MaxFileBytes {
		return TooLarge
	}
	if err := os.WriteFile(name, data, fs.FileMode(perm&0o777)); err != nil {
		return status(err)
	}
	return OK
}

func status(err error) int {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return NotExist
	case errors.Is(err, fs.ErrExist):
		return Exist
	case errors.Is(err, fs.ErrPermission):
		return Permission
	case errors.Is(err, syscall.EISDIR):
		return IsDir
	case errors.Is(err, syscall.ENOTDIR):
		return NotDir
	case errors.Is(err, syscall.EFBIG):
		return TooLarge
	case errors.Is(err, errors.ErrUnsupported):
		return Unsupported
	case errors.Is(err, fs.ErrInvalid), errors.Is(err, syscall.EINVAL):
		return Invalid
	}
	return IO
}
