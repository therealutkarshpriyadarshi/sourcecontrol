package commit

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/objects"
)

func createTestPerson(name, email string) *CommitPerson {
	person, _ := NewCommitPerson(name, email, time.Unix(1609459200, 0).UTC())
	return person
}

func TestCommitBuilder(t *testing.T) {
	author := createTestPerson("John Doe", "john@example.com")
	committer := createTestPerson("Jane Smith", "jane@example.com")

	t.Run("successful build", func(t *testing.T) {
		commit, err := NewCommitBuilder().
			Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
			Parent("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0").
			Author(author).
			Committer(committer).
			Message("Initial commit").
			Build()

		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if commit == nil {
			t.Fatal("Build() returned nil commit")
		}
		if commit.TreeSHA != "4b825dc642cb6eb9a060e54bf8d69288fbee4904" {
			t.Errorf("TreeSHA = %v", commit.TreeSHA)
		}
		if len(commit.ParentSHAs) != 1 {
			t.Errorf("ParentSHAs length = %v, want 1", len(commit.ParentSHAs))
		}
		if commit.Message != "Initial commit" {
			t.Errorf("Message = %v", commit.Message)
		}
	})

	t.Run("build with multiple parents", func(t *testing.T) {
		commit, err := NewCommitBuilder().
			Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
			Parents(
				"a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
				"b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1",
			).
			Author(author).
			Committer(committer).
			Message("Merge commit").
			Build()

		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if len(commit.ParentSHAs) != 2 {
			t.Errorf("ParentSHAs length = %v, want 2", len(commit.ParentSHAs))
		}
		if !commit.IsMergeCommit() {
			t.Error("Expected merge commit")
		}
	})

	t.Run("build without parents (initial commit)", func(t *testing.T) {
		commit, err := NewCommitBuilder().
			Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
			Author(author).
			Committer(committer).
			Message("Initial commit").
			Build()

		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if !commit.IsInitialCommit() {
			t.Error("Expected initial commit")
		}
	})

	t.Run("build fails without tree", func(t *testing.T) {
		_, err := NewCommitBuilder().
			Author(author).
			Committer(committer).
			Message("Test").
			Build()

		if err == nil {
			t.Error("Build() should fail without tree SHA")
		}
	})

	t.Run("build fails without author", func(t *testing.T) {
		_, err := NewCommitBuilder().
			Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
			Committer(committer).
			Message("Test").
			Build()

		if err == nil {
			t.Error("Build() should fail without author")
		}
	})

	t.Run("build fails without committer", func(t *testing.T) {
		_, err := NewCommitBuilder().
			Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
			Author(author).
			Message("Test").
			Build()

		if err == nil {
			t.Error("Build() should fail without committer")
		}
	})

	t.Run("build fails with invalid tree SHA", func(t *testing.T) {
		_, err := NewCommitBuilder().
			Tree("invalid").
			Author(author).
			Committer(committer).
			Message("Test").
			Build()

		if err == nil {
			t.Error("Build() should fail with invalid tree SHA")
		}
	})

	t.Run("build fails with invalid parent SHA", func(t *testing.T) {
		_, err := NewCommitBuilder().
			Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
			Parent("invalid").
			Author(author).
			Committer(committer).
			Message("Test").
			Build()

		if err == nil {
			t.Error("Build() should fail with invalid parent SHA")
		}
	})

	t.Run("build fails with nil author", func(t *testing.T) {
		_, err := NewCommitBuilder().
			Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
			Author(nil).
			Committer(committer).
			Message("Test").
			Build()

		if err == nil {
			t.Error("Build() should fail with nil author")
		}
	})
}

func TestCommit_Type(t *testing.T) {
	commit := &Commit{}
	if commit.Type() != objects.CommitType {
		t.Errorf("Type() = %v, want %v", commit.Type(), objects.CommitType)
	}
}

func TestCommit_Content(t *testing.T) {
	author := createTestPerson("John Doe", "john@example.com")
	committer := createTestPerson("Jane Smith", "jane@example.com")

	commit, _ := NewCommitBuilder().
		Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Parent("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0").
		Author(author).
		Committer(committer).
		Message("Test commit message").
		Build()

	contentObj, err := commit.Content()
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}
	content := contentObj.String()

	// Check that content contains expected parts
	if !strings.Contains(content, "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904") {
		t.Error("Content should contain tree line")
	}
	if !strings.Contains(content, "parent a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0") {
		t.Error("Content should contain parent line")
	}
	if !strings.Contains(content, "author John Doe <john@example.com>") {
		t.Error("Content should contain author line")
	}
	if !strings.Contains(content, "committer Jane Smith <jane@example.com>") {
		t.Error("Content should contain committer line")
	}
	if !strings.Contains(content, "Test commit message") {
		t.Error("Content should contain message")
	}

	// Check structure
	lines := strings.Split(content, "\n")
	if !strings.HasPrefix(lines[0], "tree ") {
		t.Error("First line should be tree")
	}
	if !strings.HasPrefix(lines[1], "parent ") {
		t.Error("Second line should be parent")
	}
	if !strings.HasPrefix(lines[2], "author ") {
		t.Error("Third line should be author")
	}
	if !strings.HasPrefix(lines[3], "committer ") {
		t.Error("Fourth line should be committer")
	}
	if lines[4] != "" {
		t.Error("Fifth line should be empty")
	}
}

func TestCommit_Serialize(t *testing.T) {
	author := createTestPerson("John Doe", "john@example.com")
	committer := createTestPerson("Jane Smith", "jane@example.com")

	commit, _ := NewCommitBuilder().
		Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Author(author).
		Committer(committer).
		Message("Test").
		Build()

	var buf bytes.Buffer
	err := commit.Serialize(&buf)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	data := buf.Bytes()
	// Check header
	if !bytes.HasPrefix(data, []byte("commit ")) {
		t.Error("Serialized data should start with 'commit '")
	}

	// Check null byte in header
	nullIndex := bytes.IndexByte(data, objects.NullByte)
	if nullIndex == -1 {
		t.Error("Serialized data should contain null byte")
	}
}

func TestParseCommit(t *testing.T) {
	author := createTestPerson("John Doe", "john@example.com")
	committer := createTestPerson("Jane Smith", "jane@example.com")

	original, _ := NewCommitBuilder().
		Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Parent("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0").
		Parent("b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1").
		Author(author).
		Committer(committer).
		Message("Test commit\n\nWith multiple lines").
		Build()

	// Serialize
	var buf bytes.Buffer
	err := original.Serialize(&buf)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Parse back
	parsed, err := ParseCommit(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseCommit() error = %v", err)
	}

	// Verify
	if parsed.TreeSHA != original.TreeSHA {
		t.Errorf("TreeSHA = %v, want %v", parsed.TreeSHA, original.TreeSHA)
	}
	if len(parsed.ParentSHAs) != len(original.ParentSHAs) {
		t.Errorf("ParentSHAs length = %v, want %v", len(parsed.ParentSHAs), len(original.ParentSHAs))
	}
	for i, parent := range parsed.ParentSHAs {
		if parent != original.ParentSHAs[i] {
			t.Errorf("Parent %d = %v, want %v", i, parent, original.ParentSHAs[i])
		}
	}
	if !parsed.Author.Equal(original.Author) {
		t.Error("Author mismatch")
	}
	if !parsed.Committer.Equal(original.Committer) {
		t.Error("Committer mismatch")
	}
	if parsed.Message != original.Message {
		t.Errorf("Message = %v, want %v", parsed.Message, original.Message)
	}
	parsedHash, err := parsed.Hash()
	if err != nil {
		t.Fatalf("parsed.Hash() error = %v", err)
	}
	originalHash, err := original.Hash()
	if err != nil {
		t.Fatalf("original.Hash() error = %v", err)
	}
	if parsedHash != originalHash {
		t.Errorf("Hash = %s, want %s", parsedHash, originalHash)
	}
}

func TestParseCommit_InvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "wrong type",
			data: []byte("blob 10\x00test data"),
		},
		{
			name: "missing tree",
			data: []byte("commit 50\x00author John <j@e.com> 123 +0000\ncommitter John <j@e.com> 123 +0000\n\nTest"),
		},
		{
			name: "missing author",
			data: []byte("commit 50\x00tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\ncommitter John <j@e.com> 123 +0000\n\nTest"),
		},
		{
			name: "missing committer",
			data: []byte("commit 50\x00tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nauthor John <j@e.com> 123 +0000\n\nTest"),
		},
		{
			name: "duplicate tree",
			data: []byte("commit 50\x00tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\ntree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nauthor J <j@e.com> 123 +0000\ncommitter J <j@e.com> 123 +0000\n\nTest"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCommit(tt.data)
			if err == nil {
				t.Error("ParseCommit() expected error, got nil")
			}
		})
	}
}

func TestCommit_Hash(t *testing.T) {
	author := createTestPerson("John Doe", "john@example.com")

	commit, _ := NewCommitBuilder().
		Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Author(author).
		Committer(author).
		Message("Test").
		Build()

	hash1, err := commit.Hash()
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	hash2, err := commit.Hash()
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	// Hash should be consistent
	if hash1 != hash2 {
		t.Error("Hash() not consistent")
	}

	// Hash should be valid (40 hex characters)
	if !hash1.IsValid() {
		t.Errorf("Hash() is not valid: %s", hash1)
	}
}

func TestCommit_IsInitialCommit(t *testing.T) {
	author := createTestPerson("John Doe", "john@example.com")

	commit, _ := NewCommitBuilder().
		Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Author(author).
		Committer(author).
		Message("Initial").
		Build()

	if !commit.IsInitialCommit() {
		t.Error("IsInitialCommit() should return true")
	}

	hash, _ := objects.NewObjectHashFromString("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0")
	commit.ParentSHAs = []objects.ObjectHash{hash}
	if commit.IsInitialCommit() {
		t.Error("IsInitialCommit() should return false")
	}
}

func TestCommit_IsMergeCommit(t *testing.T) {
	author := createTestPerson("John Doe", "john@example.com")

	commit, _ := NewCommitBuilder().
		Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Author(author).
		Committer(author).
		Message("Test").
		Build()

	if commit.IsMergeCommit() {
		t.Error("IsMergeCommit() should return false for zero parents")
	}

	hash1, _ := objects.NewObjectHashFromString("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0")
	commit.ParentSHAs = []objects.ObjectHash{hash1}
	if commit.IsMergeCommit() {
		t.Error("IsMergeCommit() should return false for one parent")
	}

	hash2, _ := objects.NewObjectHashFromString("b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1")
	commit.ParentSHAs = append(commit.ParentSHAs, hash2)
	if !commit.IsMergeCommit() {
		t.Error("IsMergeCommit() should return true for two parents")
	}
}

func TestCommit_Equal(t *testing.T) {
	author := createTestPerson("John Doe", "john@example.com")

	commit1, _ := NewCommitBuilder().
		Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Parent("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0").
		Author(author).
		Committer(author).
		Message("Test").
		Build()

	commit2, _ := NewCommitBuilder().
		Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Parent("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0").
		Author(author).
		Committer(author).
		Message("Test").
		Build()

	if !commit1.Equal(commit2) {
		t.Error("Equal commits should be equal")
	}

	commit3, _ := NewCommitBuilder().
		Tree("5b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Author(author).
		Committer(author).
		Message("Test").
		Build()

	if commit1.Equal(commit3) {
		t.Error("Different commits should not be equal")
	}

	if commit1.Equal(nil) {
		t.Error("Equal(nil) should return false")
	}
}

func TestCommit_Clone(t *testing.T) {
	author := createTestPerson("John Doe", "john@example.com")

	original, _ := NewCommitBuilder().
		Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Parent("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0").
		Author(author).
		Committer(author).
		Message("Test").
		Build()

	clone := original.Clone()

	if !original.Equal(clone) {
		t.Error("Clone should be equal to original")
	}

	// Modify clone and ensure original is unchanged
	clone.Message = "Modified"
	if original.Message == clone.Message {
		t.Error("Modifying clone should not affect original")
	}
}

func TestCommit_HasParent(t *testing.T) {
	author := createTestPerson("John Doe", "john@example.com")

	commit, _ := NewCommitBuilder().
		Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Parent("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0").
		Parent("b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1").
		Author(author).
		Committer(author).
		Message("Test").
		Build()

	if !commit.HasParent("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0") {
		t.Error("HasParent() should return true for existing parent")
	}

	if !commit.HasParent("B2C3D4E5F6A7B8C9D0E1F2A3B4C5D6E7F8A9B0C1") {
		t.Error("HasParent() should be case-insensitive")
	}

	if commit.HasParent("c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2") {
		t.Error("HasParent() should return false for non-existing parent")
	}
}

func TestCommit_ShortSHA(t *testing.T) {
	author := createTestPerson("John Doe", "john@example.com")

	commit, _ := NewCommitBuilder().
		Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Author(author).
		Committer(author).
		Message("Test").
		Build()

	shortSHA, err := commit.ShortSHA()
	if err != nil {
		t.Fatalf("ShortSHA() error = %v", err)
	}
	if shortSHA.Length() != 7 {
		t.Errorf("ShortSHA() length = %v, want 7", shortSHA.Length())
	}
}

func TestCommit_BaseObjectInterface(t *testing.T) {
	// Ensure Commit implements BaseObject interface
	var _ objects.BaseObject = (*Commit)(nil)

	author := createTestPerson("John Doe", "john@example.com")

	commit, _ := NewCommitBuilder().
		Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Author(author).
		Committer(author).
		Message("Test").
		Build()

	if commit.Type() != objects.CommitType {
		t.Errorf("Type() = %v, want %v", commit.Type(), objects.CommitType)
	}

	content, err := commit.Content()
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}
	if content.IsEmpty() {
		t.Error("Content() should not be empty")
	}

	hash, err := commit.Hash()
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if !hash.IsValid() {
		t.Errorf("Hash() is not valid: %s", hash)
	}

	size, err := commit.Size()
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if size.Int64() != int64(len(content.Bytes())) {
		t.Errorf("Size() = %v, want %v", size, len(content.Bytes()))
	}

	var buf bytes.Buffer
	err = commit.Serialize(&buf)
	if err != nil {
		t.Errorf("Serialize() error = %v", err)
	}

	str := commit.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
}

func TestValidateSHA(t *testing.T) {
	tests := []struct {
		name    string
		sha     string
		wantErr bool
	}{
		{
			name:    "valid SHA",
			sha:     "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
			wantErr: false,
		},
		{
			name:    "valid SHA uppercase",
			sha:     "4B825DC642CB6EB9A060E54BF8D69288FBEE4904",
			wantErr: false,
		},
		{
			name:    "too short",
			sha:     "4b825dc642cb6eb9a060e54bf8d69288fbee490",
			wantErr: true,
		},
		{
			name:    "too long",
			sha:     "4b825dc642cb6eb9a060e54bf8d69288fbee49041",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			sha:     "4b825dc642cb6eb9a060e54bf8d69288fbee490g",
			wantErr: true,
		},
		{
			name:    "empty",
			sha:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := objects.NewObjectHashFromString(tt.sha)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewObjectHashFromString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommit_HeaderSize(t *testing.T) {
	author := createTestPerson("John Doe", "john@example.com")

	commit, _ := NewCommitBuilder().
		Tree("4b825dc642cb6eb9a060e54bf8d69288fbee4904").
		Parent("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0").
		Author(author).
		Committer(author).
		Message("This is a long message that should not be included in header size").
		Build()

	headerSize := commit.HeaderSize()
	contentObj, err := commit.Content()
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}
	content := contentObj.String()

	// Header size should not include the message
	if headerSize >= len(content) {
		t.Error("HeaderSize() should be less than total content size")
	}

	// Header should end with double newline
	header := content[:headerSize]
	if !strings.HasSuffix(header, "\n") {
		t.Error("Header should end with newline")
	}
}
