package capability

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// Has determines if the current process has the specified capability
func Has(capability int) (bool, error) {
	var hdr unix.CapUserHeader
	var data unix.CapUserData

	hdr.Version = unix.LINUX_CAPABILITY_VERSION_3
	hdr.Pid = 0

	if err := unix.Capget(&hdr, &data); err != nil {
		return false, fmt.Errorf("capget failed: %w", err)
	}

	return (data.Permitted & (1 << uint(capability))) != 0, nil
}

// Acquire attempts to gain the specified capability for the current process
func Acquire(capability int) error {
	var hdr unix.CapUserHeader
	var data unix.CapUserData

	hdr.Version = unix.LINUX_CAPABILITY_VERSION_3
	hdr.Pid = 0

	if err := unix.Capget(&hdr, &data); err != nil {
		return fmt.Errorf("capget failed: %w", err)
	}

	data.Effective |= (1 << uint(capability))
	data.Permitted |= (1 << uint(capability))

	if err := unix.Capset(&hdr, &data); err != nil {
		return fmt.Errorf("capset failed: %w", err)
	}

	return nil
}

// Drop removes the specified capability from the current process
func Drop(capability int) error {
	var hdr unix.CapUserHeader
	var data unix.CapUserData

	hdr.Version = unix.LINUX_CAPABILITY_VERSION_3
	hdr.Pid = 0

	if err := unix.Capget(&hdr, &data); err != nil {
		return fmt.Errorf("capget failed: %w", err)
	}

	data.Effective &= ^(1 << uint(capability))
	data.Permitted &= ^(1 << uint(capability))

	if err := unix.Capset(&hdr, &data); err != nil {
		return fmt.Errorf("capset failed: %w", err)
	}

	return nil
}
