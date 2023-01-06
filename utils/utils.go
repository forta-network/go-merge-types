package utils

import "path"

// RelativePath finds the relative path with respect to the base.
func RelativePath(base, input string) string {
	return path.Join(path.Dir(base), input)
}
