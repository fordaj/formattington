package formattington

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// isIgnoredDirName reports whether discovery should prune a directory by
// name.
func isIgnoredDirName(name string) bool {
	return strings.HasPrefix(name, ".") || ignoredDirNames[name]
}

// DiscoverFiles discovers matching files under root (default: the current
// directory).
//
// extensions are dotted suffixes such as ".sh"; matching is
// case-insensitive. Hidden directories, the ignored directory names, and
// anything under them are skipped. Results are sorted and returned as
// paths relative to root so they can be appended to a formatter argv
// directly (paths with spaces stay single entries).
func DiscoverFiles(extensions []string, root string) []string {
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil
		}
		root = cwd
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil
	}

	wanted := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		wanted[strings.ToLower(ext)] = true
	}

	var found []string
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			if path == root {
				return nil
			}
			if isIgnoredDirName(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !wanted[pathSuffix(entry.Name())] {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		found = append(found, relative)
		return nil
	})

	sort.Strings(found)
	return found
}
