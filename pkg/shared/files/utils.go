package files

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
)

// ExpandPath resolves paths that include a tilde (~) to the user's home directory.
func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, path[2:]), nil
	}
	return path, nil
}

// ValidatePath checks if the given path is a valid file path for reading.
func ValidatePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path stat error: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path %q is a directory, not a file", path)
	}

	if info.Mode()&os.ModeType != 0 {
		return fmt.Errorf("path %q is not a regular file", path)
	}
	return nil
}

// GetValidatedFileName validates the given file path and returns the file name.
func GetValidatedFileName(path string) (string, error) {
	if err := ValidatePath(path); err != nil {
		return "", err
	}
	return filepath.Base(path), nil
}

// CopyDotFiles copies dot-prefixed entries from src to dst using root-scoped ops.
func CopyDotFiles(src, dst string, logger hclog.Logger) error {
	logger.Debug("copying files starting with a dot", "source", src, "destination", dst)
	srcRoot, err := os.OpenRoot(src)
	if err != nil {
		return fmt.Errorf("failed to open source root %q: %w", src, err)
	}
	defer srcRoot.Close()

	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination root %q: %w", dst, err)
	}

	dstRoot, err := os.OpenRoot(dst)
	if err != nil {
		return fmt.Errorf("failed to open destination root %q: %w", dst, err)
	}
	defer dstRoot.Close()

	files, err := fs.ReadDir(srcRoot.FS(), ".")
	if err != nil {
		return fmt.Errorf("failed to read directory %q: %w", srcRoot.Name(), err)
	}

	for _, file := range files {
		if file.Name()[0] == '.' {
			srcPath := file.Name()
			dstPath := file.Name()

			if file.IsDir() {
				if err := copyWithRoot(srcRoot, dstRoot, srcPath, dstPath); err != nil {
					logger.Error("error copying directory", "srcRoot", srcRoot.Name(), "path", srcPath, "error", err)
					return err
				}
			} else {
				if err := copyWithRoot(srcRoot, dstRoot, srcPath, dstPath); err != nil {
					logger.Error("error copying file", "srcRoot", srcRoot.Name(), "path", srcPath, "error", err)
					return err
				}
			}
		}
	}
	return nil
}

// RemoveAndRecreate removes the directory if it exists and then creates it again.
// It guarantees the target is empty before population.
func RemoveAndRecreate(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove %q: %w", path, err)
	}
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create %q: %w", path, err)
	}
	return nil
}

// Copy copies a file or directory using root paths and root-scoped operations.
func Copy(srcRootPath, dstRootPath, srcPath, destPath string) error {
	srcRoot, err := os.OpenRoot(srcRootPath)
	if err != nil {
		return fmt.Errorf("failed to open source root %q: %w", srcRootPath, err)
	}
	defer srcRoot.Close()

	dstRoot, err := os.OpenRoot(dstRootPath)
	if err != nil {
		return fmt.Errorf("failed to open destination root %q: %w", dstRootPath, err)
	}
	defer dstRoot.Close()

	return copyWithRoot(srcRoot, dstRoot, srcPath, destPath)
}

// copyWithRoot dispatches copying by source type using root-scoped operations.
func copyWithRoot(srcRoot, dstRoot *os.Root, srcPath, destPath string) error {
	srcInfo, err := srcRoot.Lstat(srcPath)
	if err != nil {
		return fmt.Errorf("failed to stat source path %q (root %q): %w", srcPath, srcRoot.Name(), err)
	}

	switch {
	case srcInfo.IsDir():
		return CopyDir(srcRoot, dstRoot, srcPath, destPath, srcInfo.Mode().Perm())
	case srcInfo.Mode()&os.ModeSymlink != 0:
		return CopySymLink(srcRoot, dstRoot, srcPath, destPath)
	default:
		return CopyFile(srcRoot, dstRoot, srcPath, destPath, srcInfo.Mode().Perm())
	}
}

// CopyFile copies a file using root-scoped paths and ensures the destination dir exists.
func CopyFile(srcRoot, dstRoot *os.Root, srcFile, destFile string, perm fs.FileMode) error {
	destDir := filepath.Dir(destFile)
	srcDirInfo, err := srcRoot.Stat(filepath.Dir(srcFile))
	if err != nil {
		return fmt.Errorf("failed to stat source directory %q (root %q): %w", filepath.Dir(srcFile), srcRoot.Name(), err)
	}
	if err := dstRoot.MkdirAll(destDir, srcDirInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("failed to create directory %q (root %q): %w", destDir, dstRoot.Name(), err)
	}

	in, err := srcRoot.Open(srcFile)
	if err != nil {
		return fmt.Errorf("failed to open source file %q (root %q): %w", srcFile, srcRoot.Name(), err)
	}
	defer in.Close()

	out, err := dstRoot.OpenFile(destFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("failed to create destination file %q (root %q): %w", destFile, dstRoot.Name(), err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy data from %q (root %q) to %q (root %q): %w", srcFile, srcRoot.Name(), destFile, dstRoot.Name(), err)
	}
	return nil
}

// CopyDir copies a directory recursively using root-scoped operations.
func CopyDir(srcRoot, dstRoot *os.Root, srcDir, destDir string, perm fs.FileMode) error {
	entries, err := fs.ReadDir(srcRoot.FS(), srcDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory %q (root %q): %w", srcDir, srcRoot.Name(), err)
	}

	if err := dstRoot.MkdirAll(destDir, perm); err != nil {
		return fmt.Errorf("failed to create destination directory %q (root %q): %w", destDir, dstRoot.Name(), err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		if err := copyWithRoot(srcRoot, dstRoot, srcPath, destPath); err != nil {
			return err
		}
	}

	return nil
}

// CopySymLink copies a symlink using root-scoped operations.
func CopySymLink(srcRoot, dstRoot *os.Root, srcLink, destLink string) error {
	linkTarget, err := srcRoot.Readlink(srcLink)
	if err != nil {
		return fmt.Errorf("failed to read symlink %q (root %q): %w", srcLink, srcRoot.Name(), err)
	}

	destDir := filepath.Dir(destLink)
	if err := dstRoot.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory %q (root %q): %w", destDir, dstRoot.Name(), err)
	}

	if destInfo, err := dstRoot.Lstat(destLink); err == nil {
		if destInfo.Mode()&os.ModeSymlink != 0 {
			if existingTarget, err := dstRoot.Readlink(destLink); err == nil && existingTarget == linkTarget {
				return nil
			}
		}
		if destInfo.IsDir() {
			// if err := dstRoot.RemoveAll(destLink); err != nil {
			// 	return fmt.Errorf("failed to remove destination directory %q: %w", destLink, err)
			// }
			return nil
		} else {
			if err := dstRoot.Remove(destLink); err != nil {
				return fmt.Errorf("failed to remove destination link %q (root %q) for source copy of %q (root %q): %w", destLink, dstRoot.Name(), srcLink, srcRoot.Name(), err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat destination %q (root %q): %w", destLink, dstRoot.Name(), err)
	}

	// check a symlink dst file exists or reachable inside root
	resolved := linkTarget
	if !filepath.IsAbs(linkTarget) {
		resolved = filepath.Clean(filepath.Join(filepath.Dir(destLink), linkTarget))
	}
	if _, err := dstRoot.Stat(resolved); err != nil {
		return fmt.Errorf("failed to validate symlink %q -> %q (root %q): %w", destLink, linkTarget, dstRoot.Name(), err)
	}

	if err := dstRoot.Symlink(linkTarget, destLink); err != nil {
		return fmt.Errorf("failed to create symlink %q -> %q (root %q): %w", destLink, linkTarget, dstRoot.Name(), err)
	}
	return nil
}

// CreateFolderIfNotExists checks if a folder exists, and if not, creates it.
func CreateFolderIfNotExists(folder string) error {
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		if err := os.MkdirAll(folder, os.ModePerm); err != nil {
			return fmt.Errorf("unable to create folder %q: %w", folder, err)
		}
	} else if err != nil {
		return fmt.Errorf("unable to check folder %q: %w", folder, err)
	}
	return nil
}

// FindByExtAndRemove walks through the directory tree rooted at root and removes files with specified extensions.
func FindByExtAndRemove(root string, exts []string) {
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access %q: %w", path, err)
		}
		ext := filepath.Ext(d.Name())
		match := false
		for _, rmExt := range exts {
			if fmt.Sprintf(".%s", rmExt) == ext {
				match = true
				break
			}
		}
		if !match {
			return nil
		}
		err = os.Remove(path)
		if err != nil {
			return fmt.Errorf("failed to remove file %q: %w", path, err)
		}
		return nil
	})
}

// WriteJsonFile writes JSON data to the specified file.
func WriteJsonFile(outputFile string, data []byte) error {
	file, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed creating file: %w", err)
	}
	defer file.Close()

	datawriter := bufio.NewWriter(file)
	defer datawriter.Flush()

	if _, err := datawriter.Write(data); err != nil {
		return fmt.Errorf("error writing data to file: %w", err)
	}

	return nil
}

func DetermineFileFullPath(path, nameTemplate string) (string, string, error) {
	// TODO: consider secure file usage
	// original := path
	// Path normalization
	// path = filepath.Clean(path)

	// Normalization check
	// if path != original {
	// 	// return "", "", fmt.Errorf("the given path %q could not be accepted due to normalization result %q", original, path)
	// }

	path, err := ExpandPath(path)
	if err != nil {
		return "", "", fmt.Errorf("failed to unwrap path %q: %w", path, err)
	}

	fileInfo, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return "", "", fmt.Errorf("failed to unwrap path %q: %w", path, err)
	}

	var fullPath, folder string
	// If file doesn't exist or no extension, treat as directory
	if err == nil && fileInfo.IsDir() || (err != nil && filepath.Ext(path) == "") {
		// It's a directory
		folder = path
		fullPath = filepath.Join(path, nameTemplate)
	} else {
		// Has extension, treat as file
		folder = filepath.Dir(path)
		fullPath = path
	}

	return fullPath, folder, nil
}
