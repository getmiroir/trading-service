//go:build windows

package keyring

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

var (
	crypt32                = windows.NewLazySystemDLL("crypt32.dll")
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
	procLocalFree          = kernel32.NewProc("LocalFree")
)

const cryptProtectLocalMachine = 0x4

func bytesToBlob(src []byte) dataBlob {
	if len(src) == 0 {
		return dataBlob{}
	}
	return dataBlob{
		cbData: uint32(len(src)),
		pbData: &src[0],
	}
}

func blobToBytes(blob dataBlob) ([]byte, error) {
	if blob.cbData == 0 || blob.pbData == nil {
		return []byte{}, nil
	}
	raw := unsafe.Slice(blob.pbData, blob.cbData)
	out := make([]byte, len(raw))
	copy(out, raw)
	procLocalFree.Call(uintptr(unsafe.Pointer(blob.pbData)))
	return out, nil
}

func dpapiEncrypt(plain []byte) ([]byte, error) {
	in := bytesToBlob(plain)
	var out dataBlob

	ret, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		uintptr(cryptProtectLocalMachine),
		uintptr(unsafe.Pointer(&out)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("CryptProtectData failed: %w", err)
	}
	return blobToBytes(out)
}

func dpapiDecrypt(cipher []byte) ([]byte, error) {
	in := bytesToBlob(cipher)
	var out dataBlob

	ret, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("CryptUnprotectData failed: %w", err)
	}
	return blobToBytes(out)
}
