//go:build linux
// +build linux

package metric

import "syscall"

func getStatfs(mountPoint string) (syscall.Statfs_t, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(mountPoint, &stat); err != nil {
		return stat, err
	}
	return stat, nil
}
