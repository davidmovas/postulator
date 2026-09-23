package wails_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type syncFake struct {
	mode     failure
	filename string
	body     []byte
}

func (f syncFake) SyncSite(context.Context, sync.SyncSiteRequest) (sync.SyncSiteResponse, error) {
	return answer[sync.SyncSiteResponse](f.mode)
}

func (f syncFake) CheckPlugin(context.Context, sync.CheckPluginRequest) (sync.CheckPluginResponse, error) {
	return answer[sync.CheckPluginResponse](f.mode)
}

func (f syncFake) PluginPackage(context.Context, sync.PluginPackageRequest) (sync.PluginPackageResponse, error) {
	if f.body == nil {
		return answer[sync.PluginPackageResponse](f.mode)
	}
	return sync.PluginPackageResponse{Filename: f.filename, Bytes: f.body}, nil
}

func TestSyncServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewSyncService(zap.NewNop(), ready[wails.SyncUseCase](syncFake{})), []string{
		"CheckPlugin", "SavePluginPackage", "SyncSite",
	})
	assertEveryMethodConverts(t, wails.NewSyncService(zap.NewNop(), ready[wails.SyncUseCase](syncFake{mode: missing})), missingBody, "SavePluginPackage")
	assertEveryMethodConverts(t, wails.NewSyncService(zap.NewNop(), ready[wails.SyncUseCase](syncFake{mode: panicking})), panicBody, "SavePluginPackage")
}

func TestSavePluginPackageWritesTheArchive(t *testing.T) {
	t.Parallel()

	packaged := syncFake{filename: "postulator-companion.zip", body: []byte("PK\x03\x04archive")}
	home := t.TempDir()

	cases := []struct {
		name string
		path func() string
		want func(home string) string
	}{
		{
			name: "a file path is written as it stands",
			path: func() string { return filepath.Join(home, "chosen.zip") },
			want: func(home string) string { return filepath.Join(home, "chosen.zip") },
		},
		{
			name: "a directory takes the package filename",
			path: func() string { return home },
			want: func(home string) string { return filepath.Join(home, "postulator-companion.zip") },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := wails.NewSyncService(zap.NewNop(), ready[wails.SyncUseCase](packaged))

			saved, err := service.SavePluginPackage(context.Background(), wails.SavePluginPackageRequest{Path: tc.path()})
			if err != nil {
				t.Fatalf("SavePluginPackage: %v", err)
			}
			if saved.Path != tc.want(home) {
				t.Errorf("Path = %q, want %q", saved.Path, tc.want(home))
			}
			if saved.Filename != packaged.filename || saved.Bytes != len(packaged.body) {
				t.Errorf("response = %+v, want the package name and its size", saved)
			}

			written, readErr := os.ReadFile(saved.Path)
			if readErr != nil {
				t.Fatalf("read back the archive: %v", readErr)
			}
			if !bytes.Equal(written, packaged.body) {
				t.Errorf("archive = %q, want %q", written, packaged.body)
			}
		})
	}
}

func TestSavePluginPackageRefusesADestinationItCannotUse(t *testing.T) {
	t.Parallel()

	packaged := syncFake{filename: "postulator-companion.zip", body: []byte("PK\x03\x04archive")}

	cases := []struct {
		name    string
		useCase syncFake
		path    string
		want    string
	}{
		{
			name:    "no path",
			useCase: packaged,
			path:    "   ",
			want:    `{"code":"INVALID","message":"the destination path must not be empty","details":{"field":"path"}}`,
		},
		{
			name:    "the packager fails",
			useCase: syncFake{mode: missing},
			path:    filepath.Join(t.TempDir(), "companion.zip"),
			want:    missingBody,
		},
		{
			name:    "the destination directory does not exist",
			useCase: packaged,
			path:    filepath.Join(t.TempDir(), "absent", "companion.zip"),
			want:    `{"code":"INVALID","message":"the companion plugin archive could not be written"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := wails.NewSyncService(zap.NewNop(), ready[wails.SyncUseCase](tc.useCase))

			_, err := service.SavePluginPackage(context.Background(), wails.SavePluginPackageRequest{Path: tc.path})
			if err == nil {
				t.Fatal("SavePluginPackage returned no error")
			}

			got := string(wails.MarshalError(err))
			if len(got) < len(tc.want) || got[:len(tc.want)] != tc.want {
				t.Fatalf("MarshalError() = %s, want it to start with %s", got, tc.want)
			}
		})
	}
}
