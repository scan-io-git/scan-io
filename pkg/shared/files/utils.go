package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
)

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
				if err := Copy(srcPath, dst); err != nil {
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
