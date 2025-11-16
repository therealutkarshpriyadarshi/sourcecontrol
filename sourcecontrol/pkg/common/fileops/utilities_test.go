package fileops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
)

func TestExists(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("file exists", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		exists, err := Exists(scpath.AbsolutePath(filePath))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected file to exist")
		}
	})

	t.Run("file does not exist", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "nonexistent.txt")

		exists, err := Exists(scpath.AbsolutePath(filePath))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if exists {
			t.Error("expected file to not exist")
		}
	})

	t.Run("directory exists", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "testdir")
		if err := os.Mkdir(dirPath, 0755); err != nil {
			t.Fatal(err)
		}

		exists, err := Exists(scpath.AbsolutePath(dirPath))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected directory to exist")
		}
	})
}

func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("create new directory", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "newdir")

		if err := EnsureDir(scpath.AbsolutePath(dirPath)); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		info, err := os.Stat(dirPath)
		if err != nil {
			t.Errorf("directory was not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected path to be a directory")
		}
	})

	t.Run("create nested directories", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "a", "b", "c")

		if err := EnsureDir(scpath.AbsolutePath(dirPath)); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		info, err := os.Stat(dirPath)
		if err != nil {
			t.Errorf("nested directories were not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected path to be a directory")
		}
	})

	t.Run("directory already exists", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "existing")
		if err := os.Mkdir(dirPath, 0755); err != nil {
			t.Fatal(err)
		}

		if err := EnsureDir(scpath.AbsolutePath(dirPath)); err != nil {
			t.Errorf("unexpected error when directory exists: %v", err)
		}
	})
}

func TestEnsureParentDir(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("create parent directories for file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "parent", "child", "file.txt")

		if err := EnsureParentDir(scpath.AbsolutePath(filePath)); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		parentDir := filepath.Dir(filePath)
		info, err := os.Stat(parentDir)
		if err != nil {
			t.Errorf("parent directory was not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected parent to be a directory")
		}
	})
}

func TestReadString(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("read existing file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "test.txt")
		content := "  hello world  \n"
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result, err := ReadString(scpath.AbsolutePath(filePath))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != "hello world" {
			t.Errorf("expected 'hello world', got '%s'", result)
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "nonexistent.txt")

		result, err := ReadString(scpath.AbsolutePath(filePath))
		if err != nil {
			t.Errorf("unexpected error for non-existent file: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})
}

func TestReadStringStrict(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("read existing file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "test.txt")
		content := "  hello world  \n"
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result, err := ReadStringStrict(scpath.AbsolutePath(filePath))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != "hello world" {
			t.Errorf("expected 'hello world', got '%s'", result)
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "nonexistent.txt")

		_, err := ReadStringStrict(scpath.AbsolutePath(filePath))
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})
}

func TestReadBytes(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("read existing file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "test.txt")
		content := []byte{0x01, 0x02, 0x03}
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatal(err)
		}

		result, err := ReadBytes(scpath.AbsolutePath(filePath))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 3 || result[0] != 0x01 {
			t.Error("content mismatch")
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "nonexistent.txt")

		result, err := ReadBytes(scpath.AbsolutePath(filePath))
		if err != nil {
			t.Errorf("unexpected error for non-existent file: %v", err)
		}
		if result != nil {
			t.Error("expected nil for non-existent file")
		}
	})
}

func TestReadBytesStrict(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("read existing file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "test.txt")
		content := []byte{0x01, 0x02, 0x03}
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatal(err)
		}

		result, err := ReadBytesStrict(scpath.AbsolutePath(filePath))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 3 || result[0] != 0x01 {
			t.Error("content mismatch")
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "nonexistent.txt")

		_, err := ReadBytesStrict(scpath.AbsolutePath(filePath))
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})
}

func TestWriteConfig(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("write config file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "config.txt")
		content := []byte("test content")

		if err := WriteConfig(scpath.AbsolutePath(filePath), content); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify file exists and has correct content
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read written file: %v", err)
		}
		if string(data) != "test content" {
			t.Errorf("content mismatch: got '%s'", string(data))
		}

		// Verify permissions (0644 on Unix, 0666 on Windows)
		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatal(err)
		}
		mode := info.Mode().Perm()
		// Windows doesn't support Unix permissions, so check is more relaxed
		if mode != 0644 && mode != 0666 {
			t.Logf("Note: got permissions %o (expected 0644 on Unix, 0666 on Windows)", mode)
		}
	})

	t.Run("write config with nested path", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "nested", "dir", "config.txt")
		content := []byte("test content")

		if err := WriteConfig(scpath.AbsolutePath(filePath), content); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(filePath); err != nil {
			t.Errorf("file was not created: %v", err)
		}
	})
}

func TestWriteReadOnly(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("write read-only file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "readonly.txt")
		content := []byte("immutable content")

		if err := WriteReadOnly(scpath.AbsolutePath(filePath), content); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify file exists and has correct content
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read written file: %v", err)
		}
		if string(data) != "immutable content" {
			t.Errorf("content mismatch: got '%s'", string(data))
		}

		// Verify permissions (0444)
		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatal(err)
		}
		mode := info.Mode().Perm()
		if mode != 0444 {
			t.Errorf("expected permissions 0444, got %o", mode)
		}
	})
}

func TestWriteConfigString(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("write string content", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "string.txt")
		content := "string content"

		if err := WriteConfigString(scpath.AbsolutePath(filePath), content); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify file exists and has correct content
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read written file: %v", err)
		}
		if string(data) != "string content" {
			t.Errorf("content mismatch: got '%s'", string(data))
		}
	})
}

func TestSafeRemove(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("remove existing file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "remove.txt")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		if err := SafeRemove(scpath.AbsolutePath(filePath)); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify file is removed
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Error("file was not removed")
		}
	})

	t.Run("remove non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "nonexistent.txt")

		if err := SafeRemove(scpath.AbsolutePath(filePath)); err != nil {
			t.Errorf("unexpected error for non-existent file: %v", err)
		}
	})
}

func TestIsDirectory(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("check directory", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "testdir")
		if err := os.Mkdir(dirPath, 0755); err != nil {
			t.Fatal(err)
		}

		isDir, err := IsDirectory(scpath.AbsolutePath(dirPath))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !isDir {
			t.Error("expected path to be a directory")
		}
	})

	t.Run("check file (not directory)", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "file.txt")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		isDir, err := IsDirectory(scpath.AbsolutePath(filePath))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if isDir {
			t.Error("expected path to not be a directory")
		}
	})

	t.Run("check non-existent path", func(t *testing.T) {
		path := filepath.Join(tempDir, "nonexistent")

		isDir, err := IsDirectory(scpath.AbsolutePath(path))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if isDir {
			t.Error("expected non-existent path to not be a directory")
		}
	})
}

func TestIsFile(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("check file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "file.txt")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		isFile, err := IsFile(scpath.AbsolutePath(filePath))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !isFile {
			t.Error("expected path to be a file")
		}
	})

	t.Run("check directory (not file)", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "testdir")
		if err := os.Mkdir(dirPath, 0755); err != nil {
			t.Fatal(err)
		}

		isFile, err := IsFile(scpath.AbsolutePath(dirPath))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if isFile {
			t.Error("expected path to not be a file")
		}
	})

	t.Run("check non-existent path", func(t *testing.T) {
		path := filepath.Join(tempDir, "nonexistent")

		isFile, err := IsFile(scpath.AbsolutePath(path))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if isFile {
			t.Error("expected non-existent path to not be a file")
		}
	})
}
