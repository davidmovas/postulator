package sqlite

import (
	"encoding/hex"
	"net/url"
	"path/filepath"
	"strconv"
)

const (
	keyLength         = 32
	busyTimeoutMillis = 5000
)

func dsn(path string, key []byte, readOnly bool) string {
	query := make(url.Values)
	if key != nil {
		query.Set("vfs", "adiantum")
		query.Set("hexkey", hex.EncodeToString(key))
	}
	if readOnly {
		query.Set("mode", "ro")
	} else {
		query.Set("_txlock", "immediate")
	}

	query.Add("_pragma", "busy_timeout("+strconv.Itoa(busyTimeoutMillis)+")")
	query.Add("_pragma", "foreign_keys(1)")
	if !readOnly {
		query.Add("_pragma", "journal_mode(WAL)")
		query.Add("_pragma", "synchronous(NORMAL)")
	}

	location := url.URL{Path: filepath.ToSlash(path)}
	return "file:" + location.EscapedPath() + "?" + query.Encode()
}
