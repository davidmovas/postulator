package dpapi

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var entropy = [...]byte{
	0x50, 0x6f, 0x73, 0x74, 0x75, 0x6c, 0x61, 0x74,
	0x6f, 0x72, 0x2f, 0x76, 0x32, 0x2f, 0x64, 0x70,
	0x61, 0x70, 0x69, 0x2f, 0x6d, 0x61, 0x73, 0x74,
	0x65, 0x72, 0x2d, 0x6b, 0x65, 0x79, 0x2f, 0x31,
}

func Protect(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New(errors.Invalid, "nothing to protect")
	}

	in := blob(plaintext)
	salt := applicationEntropy()
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, &salt, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "protect the buffer")
	}
	return collect(&out)
}

func Unprotect(protected []byte) ([]byte, error) {
	if len(protected) == 0 {
		return nil, errors.New(errors.Invalid, "nothing to unprotect")
	}

	in := blob(protected)
	salt := applicationEntropy()
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, &salt, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, errors.Wrap(err, errors.Invalid, "unprotect the buffer")
	}
	return collect(&out)
}

func applicationEntropy() windows.DataBlob {
	return windows.DataBlob{Size: uint32(len(entropy)), Data: &entropy[0]}
}

func blob(data []byte) windows.DataBlob {
	return windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
}

func collect(out *windows.DataBlob) ([]byte, error) {
	result := make([]byte, out.Size)
	copy(result, unsafe.Slice(out.Data, out.Size))

	handle, err := windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "release the protected buffer")
	}
	if handle != 0 {
		return nil, errors.New(errors.Internal, "release the protected buffer")
	}
	return result, nil
}
