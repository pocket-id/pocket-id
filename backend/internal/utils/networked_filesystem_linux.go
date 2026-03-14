//go:build linux

package utils

import (
	"fmt"
	"syscall"
)

// Filesystem magic values from Linux's include/uapi/linux/magic.h, used by statfs(2).
const (
	nfsSuperMagic  = 0x6969
	smbSuperMagic  = 0x517b
	cifsSuperMagic = 0xff534d42
	fuseSuperMagic = 0x65735546
)

// IsNetworkedFileSystem reports whether path is on a filesystem that is known to be unsafe for SQLite, specifically NFS, SMB/CIFS, or FUSE mounts.
func IsNetworkedFileSystem(path string) (bool, error) {
	var statfs syscall.Statfs_t
	err := syscall.Statfs(path, &statfs)
	if err != nil {
		return false, fmt.Errorf("error executing statfs syscall: %w", err)
	}

	// Statfs_t.Type is arch-dependent (for example, int32 on some systems and int64 on others).
	// Normalize through uint32 first so signed values still preserve the Linux bit pattern for magic numbers such as CIFS (0xff534d42), then compare in a wide unsigned form.
	//nolint:gosec
	switch uint64(uint32(statfs.Type)) {
	case nfsSuperMagic, smbSuperMagic, cifsSuperMagic, fuseSuperMagic:
		return true, nil
	default:
		return false, nil
	}
}
