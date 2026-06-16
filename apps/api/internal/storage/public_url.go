package storage

import (
	"net/url"
	"strings"
)

func buildPublicURL(publicBase, username, slug string) string {
	base := strings.TrimRight(strings.TrimSpace(publicBase), "/")
	return base + "/@" + url.PathEscape(username) + "/" + url.PathEscape(slug)
}
