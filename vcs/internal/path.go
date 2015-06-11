package internal

import "strings"

// Rel strips the leading "/" prefix from the path string, effectively turning
// an absolute path into one relative to the root directory. A path that is just
// "/" is treated specially, returning just ".".
func Rel(path string) string {
	if path == "/" {
		return "."
	}
	return strings.TrimPrefix(path, "/")
}
