//go:build !linux

package utils

// IsNetworkedFileSystem returns false on non-Linux systems because this detection is only used for Linux-specific statfs(2) filesystem magic values.
func IsNetworkedFileSystem(string) (bool, error) {
	return false, nil
}
