package wp

import "github.com/davidmovas/postulator/internal/domain/pagemap"

func NormalizePath(rawPath string) (string, error) {
	return pagemap.NormalizePath(rawPath)
}

func InternalPath(siteHost, href string) (string, bool) {
	return pagemap.InternalPath(href, siteHost)
}
