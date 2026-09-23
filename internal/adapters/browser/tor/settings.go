package tor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/settings"
)

var pathSetting = settings.String("browser.torPath", "", settings.Check(func(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if !filepath.IsAbs(trimmed) {
		return fmt.Errorf("the path to Tor Browser must be absolute")
	}

	info, err := os.Stat(trimmed)
	if err != nil {
		return fmt.Errorf("no file sits at that path")
	}
	if info.IsDir() {
		return fmt.Errorf("the path must name the executable, not the folder holding it")
	}
	return nil
}))

func Path(values *settings.Values) string {
	return strings.TrimSpace(pathSetting.Get(values))
}
