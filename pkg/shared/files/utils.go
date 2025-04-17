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
		return fmt.Errorf("path '%s' is a directory, not a file", path)
	}

	if info.Mode()&os.ModeType != 0 {
		return fmt.Errorf("path '%s' is not a regular file", path)
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

// CopyDotFiles copies files and directories starting with a dot from src to dst.
func CopyDotFiles(src, dst string, logger hclog.Logger) error {
	logger.Debug("copying files starting with a dot", "source", src, "destination", dst)
	files, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", src, err)
	}

	for _, file := range files {
		if file.Name()[0] == '.' {
			srcPath := filepath.Join(src, file.Name())
			dstPath := filepath.Join(dst, file.Name())

			if file.IsDir() {
				if err := Copy(srcPath, dstPath); err != nil {
					logger.Error("error copying directory", "path", srcPath, "error", err)
					return err
				}
			} else {
				if err := Copy(srcPath, dstPath); err != nil {
					logger.Error("error copying file", "path", srcPath, "error", err)
					return err
				}
			}
		}
	}
	return nil
}

// Copy determines the type of source (file, directory, or symlink) and copies it accordingly.
func Copy(srcPath, destPath string) error {
	srcInfo, err := os.Lstat(srcPath)
	if err != nil {
		return fmt.Errorf("failed to stat source path %s: %w", srcPath, err)
	}

	switch {
	case srcInfo.IsDir():
		return CopyDir(srcPath, destPath)
	case srcInfo.Mode()&os.ModeSymlink != 0:
		return CopySymLink(srcPath, destPath)
	default:
		return CopyFile(srcPath, destPath)
	}
}

// CopyFile copies a file from srcFile to destFile.
func CopyFile(srcFile, destFile string) error {
	destDir := filepath.Dir(destFile)
	if err := CreateFolderIfNotExists(destDir); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	in, err := os.Open(srcFile)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", srcFile, err)
	}
	defer in.Close()

	out, err := os.Create(destFile)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", destFile, err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy data from %s to %s: %w", srcFile, destFile, err)
	}
	return nil
}

// CopyDir copies a directory from srcDir to destDir recursively.
func CopyDir(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory %s: %w", srcDir, err)
	}

	if err := CreateFolderIfNotExists(destDir); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		if err := Copy(srcPath, destPath); err != nil {
			return err
		}
	}

	return nil
}

// CopySymLink copies a symbolic link from srcLink to destLink.
func CopySymLink(srcLink, destLink string) error {
	linkTarget, err := os.Readlink(srcLink)
	if err != nil {
		return fmt.Errorf("failed to read symlink %s: %w", srcLink, err)
	}

	if err := os.Symlink(linkTarget, destLink); err != nil {
		return fmt.Errorf("failed to create symlink %s -> %s: %w", destLink, linkTarget, err)
	}
	return nil
}

// CreateFolderIfNotExists checks if a folder exists, and if not, creates it.
func CreateFolderIfNotExists(folder string) error {
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		if err := os.MkdirAll(folder, os.ModePerm); err != nil {
			return fmt.Errorf("unable to create folder %s: %w", folder, err)
		}
	} else if err != nil {
		return fmt.Errorf("unable to check folder %s: %w", folder, err)
	}
	return nil
}

// FindByExtAndRemove walks through the directory tree rooted at root and removes files with specified extensions.
func FindByExtAndRemove(root string, exts []string) {
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access %s: %w", path, err)
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
			return fmt.Errorf("failed to remove file %s: %w", path, err)
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
	// 	// return "", "", fmt.Errorf("the given path '%s' could not be accepted due to normalization result '%s'", original, path)
	// }

	fileInfo, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return "", "", fmt.Errorf("failed to stat path '%s': %w", path, err)
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
