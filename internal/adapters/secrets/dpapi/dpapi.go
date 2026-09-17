package dpapi

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func Protect(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New(errors.Invalid, "nothing to protect")
	}

	in := blob(plaintext)
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "protect the buffer")
	}
	return collect(&out)
}

func Unprotect(protected []byte) ([]byte, error) {
	if len(protected) == 0 {
		return nil, errors.New(errors.Invalid, "nothing to unprotect")
	}

	in := blob(protected)
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "unprotect the buffer")
	}
	return collect(&out)
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
