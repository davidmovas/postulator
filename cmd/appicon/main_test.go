package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()

	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}

	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		t.Fatal("the generator test must run inside the module")
	}
	return filepath.Dir(gomod)
}

func committed(t *testing.T, path string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}

func samePixels(t *testing.T, path string, got image.Image, want *image.NRGBA) {
	t.Helper()

	if got.Bounds() != want.Bounds() {
		t.Fatalf("%s covers %v, want %v", path, got.Bounds(), want.Bounds())
	}
	for y := want.Bounds().Min.Y; y < want.Bounds().Max.Y; y++ {
		for x := want.Bounds().Min.X; x < want.Bounds().Max.X; x++ {
			if got.At(x, y) != want.At(x, y) {
				t.Fatalf("%s differs at %d,%d: %v, want %v; run task icons", path, x, y, got.At(x, y), want.At(x, y))
			}
		}
	}
}

func TestTheCommittedIconCarriesEveryDeclaredSize(t *testing.T) {
	t.Parallel()

	entries, err := decodeICO(committed(t, windowsIcon))
	if err != nil {
		t.Fatalf("decode %s: %v", windowsIcon, err)
	}

	sizes := make([]int, 0, len(entries))
	for size := range entries {
		sizes = append(sizes, size)
	}
	slices.Sort(sizes)

	if !slices.Equal(sizes, iconSizes) {
		t.Fatalf("%s carries %v, want %v; run task icons", windowsIcon, sizes, iconSizes)
	}
	for _, size := range iconSizes {
		samePixels(t, windowsIcon, entries[size], render(size))
	}
}

func TestTheCommittedImagesAreTheCurrentMark(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path string
		size int
	}{
		{path: plateIcon, size: plateIconSize},
		{path: windowIcon, size: windowIconSize},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()

			decoded, err := png.Decode(bytes.NewReader(committed(t, tc.path)))
			if err != nil {
				t.Fatalf("decode %s: %v", tc.path, err)
			}
			samePixels(t, tc.path, decoded, render(tc.size))
		})
	}
}

func TestTheCommittedVectorIsTheCurrentMark(t *testing.T) {
	t.Parallel()

	if got := committed(t, webMark); !bytes.Equal(got, vector()) {
		t.Fatalf("%s is stale; run task icons", webMark)
	}
}
