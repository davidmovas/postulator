package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
)

const directoryEntrySize = 16

func encodePNG(canvas *image.NRGBA) ([]byte, error) {
	var out bytes.Buffer
	if err := png.Encode(&out, canvas); err != nil {
		return nil, fmt.Errorf("encode %dpx: %w", canvas.Bounds().Dx(), err)
	}
	return out.Bytes(), nil
}

func encodeICO(sizes []int) ([]byte, error) {
	payloads := make([][]byte, 0, len(sizes))
	for _, size := range sizes {
		encoded, err := encodePNG(render(size))
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, encoded)
	}

	header := make([]byte, 6+directoryEntrySize*len(sizes))
	binary.LittleEndian.PutUint16(header[2:4], 1)
	binary.LittleEndian.PutUint16(header[4:6], uint16(len(sizes)))

	offset := len(header)
	for index, size := range sizes {
		at := 6 + directoryEntrySize*index
		header[at] = uint8(size % 256)
		header[at+1] = uint8(size % 256)
		binary.LittleEndian.PutUint16(header[at+4:at+6], 1)
		binary.LittleEndian.PutUint16(header[at+6:at+8], 32)
		binary.LittleEndian.PutUint32(header[at+8:at+12], uint32(len(payloads[index])))
		binary.LittleEndian.PutUint32(header[at+12:at+16], uint32(offset))
		offset += len(payloads[index])
	}

	var out bytes.Buffer
	out.Write(header)
	for _, payload := range payloads {
		out.Write(payload)
	}
	return out.Bytes(), nil
}

func decodeICO(raw []byte) (map[int]image.Image, error) {
	if len(raw) < 6 {
		return nil, fmt.Errorf("an icon needs a header, got %d bytes", len(raw))
	}
	if binary.LittleEndian.Uint16(raw[2:4]) != 1 {
		return nil, fmt.Errorf("the file is not an icon")
	}

	count := int(binary.LittleEndian.Uint16(raw[4:6]))
	if len(raw) < 6+directoryEntrySize*count {
		return nil, fmt.Errorf("the directory names %d entries the file does not carry", count)
	}

	entries := make(map[int]image.Image, count)
	for index := range count {
		at := 6 + directoryEntrySize*index
		length := int(binary.LittleEndian.Uint32(raw[at+8 : at+12]))
		offset := int(binary.LittleEndian.Uint32(raw[at+12 : at+16]))
		if offset+length > len(raw) {
			return nil, fmt.Errorf("entry %d runs past the end of the file", index)
		}

		decoded, err := png.Decode(bytes.NewReader(raw[offset : offset+length]))
		if err != nil {
			return nil, fmt.Errorf("decode entry %d: %w", index, err)
		}
		entries[decoded.Bounds().Dx()] = decoded
	}
	return entries, nil
}
