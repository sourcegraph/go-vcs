package util

// Rel strips the leading "/" prefix from the path string, effectively turning
// an absolute path into one relative to the root directory.
func Rel(path string) string {
	if len(path) > 0 && path[0] == '/' {
		return path[1:]
	}
	return path
}
