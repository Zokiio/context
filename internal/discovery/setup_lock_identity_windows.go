package discovery

import (
	"encoding/binary"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func setupLockIdentity(file *os.File, _ os.FileInfo) (setupLockID, error) {
	// FILE_ID_INFO supports the full 128-bit identifier used by ReFS as well
	// as NTFS. Refuse the write if this filesystem cannot provide the identity.
	var info struct {
		volume uint64
		id     [16]byte
	}
	if err := windows.GetFileInformationByHandleEx(windows.Handle(file.Fd()), windows.FileIdInfo, (*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		return setupLockID{}, err
	}
	return setupLockID{info.volume, binary.BigEndian.Uint64(info.id[:8]), binary.BigEndian.Uint64(info.id[8:])}, nil
}
