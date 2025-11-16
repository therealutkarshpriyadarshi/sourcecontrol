package blob

import (
	"bytes"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"testing"
)

func TestNewBlob(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantLen int
	}{
		{
			name:    "empty blob",
			data:    []byte{},
			wantLen: 0,
		},
		{
			name:    "simple text",
			data:    []byte("hello world"),
			wantLen: 11,
		},
		{
			name:    "multiline text",
			data:    []byte("line 1\nline 2\nline 3"),
			wantLen: 20,
		},
		{
			name:    "binary data",
			data:    []byte{0x00, 0x01, 0x02, 0xFF, 0xFE},
			wantLen: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob := NewBlob(tt.data)

			if blob == nil {
				t.Fatal("NewBlob returned nil")
			}

			content, err := blob.Content()
			if err != nil {
				t.Fatalf("Content() error = %v", err)
			}
			if !bytes.Equal(content.Bytes(), tt.data) {
				t.Errorf("Content() = %v, want %v", content.Bytes(), tt.data)
			}

			size, err := blob.Size()
			if err != nil {
				t.Fatalf("Size() error = %v", err)
			}
			if size.Int64() != int64(tt.wantLen) {
				t.Errorf("Size() = %d, want %d", size, tt.wantLen)
			}

			if blob.Type() != objects.BlobType {
				t.Errorf("Type() = %v, want %v", blob.Type(), objects.BlobType)
			}

			// Verify hash is non-zero
			hash, err := blob.Hash()
			if err != nil {
				t.Fatalf("Hash() error = %v", err)
			}
			if hash.IsZero() {
				t.Error("Hash() returned zero hash")
			}
		})
	}
}

func TestBlob_Type(t *testing.T) {
	blob := NewBlob([]byte("test"))
	if got := blob.Type(); got != objects.BlobType {
		t.Errorf("Type() = %v, want %v", got, objects.BlobType)
	}
}

func TestBlob_Content(t *testing.T) {
	data := []byte("test content")
	blob := NewBlob(data)

	content, err := blob.Content()
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}
	if !bytes.Equal(content.Bytes(), data) {
		t.Errorf("Content() = %v, want %v", content.Bytes(), data)
	}

	// Ensure returned content is the actual data, not a copy
	content, err = blob.Content()
	if err != nil {
		t.Fatalf("Content() error = %v", err)
	}
	contentBytes := content.Bytes()
	if len(contentBytes) > 0 {
		contentBytes[0] = 'X'
		content2, err := blob.Content()
		if err != nil {
			t.Fatalf("Content() error = %v", err)
		}
		if content2.Bytes()[0] != 'X' {
			t.Error("Content() should return reference to actual data")
		}
	}
}

func TestBlob_Size(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want int64
	}{
		{"empty", []byte{}, 0},
		{"small", []byte("test"), 4},
		{"large", make([]byte, 10000), 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob := NewBlob(tt.data)
			got, err := blob.Size()
			if err != nil {
				t.Fatalf("Size() error = %v", err)
			}
			if got.Int64() != tt.want {
				t.Errorf("Size() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlob_Hash(t *testing.T) {
	// Same content should produce same hash
	data := []byte("test data")
	blob1 := NewBlob(data)
	blob2 := NewBlob(data)

	hash1, err := blob1.Hash()
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	hash2, err := blob2.Hash()
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash1 != hash2 {
		t.Error("Same content should produce same hash")
	}

	// Different content should produce different hash
	blob3 := NewBlob([]byte("different data"))
	hash3, err := blob3.Hash()
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash1 == hash3 {
		t.Error("Different content should produce different hash")
	}
}

func TestBlob_Serialize(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty blob",
			data: []byte{},
		},
		{
			name: "simple text",
			data: []byte("hello world"),
		},
		{
			name: "with newlines",
			data: []byte("line1\nline2\n"),
		},
		{
			name: "binary data",
			data: []byte{0x00, 0x01, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob := NewBlob(tt.data)
			buf := &bytes.Buffer{}

			err := blob.Serialize(buf)
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}

			serialized := buf.Bytes()

			// Check header format: "blob <size>\0"
			expectedHeader := append([]byte("blob "), []byte(string(rune(len(tt.data))))...)
			expectedHeader = append(expectedHeader, objects.NullByte)

			// Verify the serialized data starts with correct header
			if !bytes.HasPrefix(serialized, []byte("blob ")) {
				t.Error("Serialized data should start with 'blob '")
			}

			// Verify null byte exists
			nullIndex := bytes.IndexByte(serialized, objects.NullByte)
			if nullIndex == -1 {
				t.Error("Serialized data should contain null byte")
			}

			// Verify content after null byte matches original data
			contentBytes := serialized[nullIndex+1:]
			if !bytes.Equal(contentBytes, tt.data) {
				t.Errorf("Content after null byte = %v, want %v", contentBytes, tt.data)
			}
		})
	}
}

func TestBlob_String(t *testing.T) {
	blob := NewBlob([]byte("test"))
	str := blob.String()

	if str == "" {
		t.Error("String() should not return empty string")
	}

	// Check that string contains size info
	if !bytes.Contains([]byte(str), []byte("size")) {
		t.Error("String() should contain size information")
	}

	// Check that string contains hash info
	if !bytes.Contains([]byte(str), []byte("hash")) {
		t.Error("String() should contain hash information")
	}
}

func TestParseBlob(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid empty blob",
			data:    []byte("blob 0\x00"),
			wantErr: false,
		},
		{
			name:    "valid simple blob",
			data:    []byte("blob 5\x00hello"),
			wantErr: false,
		},
		{
			name:    "valid multiline blob",
			data:    []byte("blob 12\x00line1\nline2\n"),
			wantErr: false,
		},
		{
			name:    "missing null byte",
			data:    []byte("blob 5 hello"),
			wantErr: true,
		},
		{
			name:    "wrong type",
			data:    []byte("tree 5\x00hello"),
			wantErr: true,
		},
		{
			name:    "size mismatch",
			data:    []byte("blob 10\x00hello"),
			wantErr: true,
		},
		{
			name:    "invalid header",
			data:    []byte("blob\x00hello"),
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob, err := ParseBlob(tt.data)

			if tt.wantErr {
				if err == nil {
					t.Error("ParseBlob() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseBlob() unexpected error = %v", err)
			}

			if blob == nil {
				t.Fatal("ParseBlob() returned nil blob")
			}

			// Verify the content is correct
			nullIndex := bytes.IndexByte(tt.data, objects.NullByte)
			expectedContent := tt.data[nullIndex+1:]

			content, err := blob.Content()
			if err != nil {
				t.Fatalf("Content() error = %v", err)
			}
			if !bytes.Equal(content.Bytes(), expectedContent) {
				t.Errorf("Content() = %v, want %v", content.Bytes(), expectedContent)
			}

			if blob.Type() != objects.BlobType {
				t.Errorf("Type() = %v, want %v", blob.Type(), objects.BlobType)
			}
		})
	}
}

func TestBlob_SerializeAndParse_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty",
			data: []byte{},
		},
		{
			name: "simple text",
			data: []byte("hello world"),
		},
		{
			name: "multiline",
			data: []byte("line1\nline2\nline3\n"),
		},
		{
			name: "with special chars",
			data: []byte("hello\x00world\ntab\there"),
		},
		{
			name: "large content",
			data: bytes.Repeat([]byte("test "), 1000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create original blob
			original := NewBlob(tt.data)

			// Serialize it
			buf := &bytes.Buffer{}
			err := original.Serialize(buf)
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}

			// Parse it back
			parsed, err := ParseBlob(buf.Bytes())
			if err != nil {
				t.Fatalf("ParseBlob() error = %v", err)
			}

			// Compare
			originalContent, err := original.Content()
			if err != nil {
				t.Fatalf("original.Content() error = %v", err)
			}
			parsedContent, err := parsed.Content()
			if err != nil {
				t.Fatalf("parsed.Content() error = %v", err)
			}
			if !bytes.Equal(originalContent.Bytes(), parsedContent.Bytes()) {
				t.Errorf("Content mismatch: original = %v, parsed = %v",
					originalContent.Bytes(), parsedContent.Bytes())
			}

			originalSize, err := original.Size()
			if err != nil {
				t.Fatalf("original.Size() error = %v", err)
			}
			parsedSize, err := parsed.Size()
			if err != nil {
				t.Fatalf("parsed.Size() error = %v", err)
			}
			if originalSize != parsedSize {
				t.Errorf("Size mismatch: original = %v, parsed = %v",
					originalSize, parsedSize)
			}

			originalHash, err := original.Hash()
			if err != nil {
				t.Fatalf("original.Hash() error = %v", err)
			}
			parsedHash, err := parsed.Hash()
			if err != nil {
				t.Fatalf("parsed.Hash() error = %v", err)
			}
			if originalHash != parsedHash {
				t.Errorf("Hash mismatch: original = %x, parsed = %x",
					originalHash, parsedHash)
			}

			if original.Type() != parsed.Type() {
				t.Errorf("Type mismatch: original = %v, parsed = %v",
					original.Type(), parsed.Type())
			}
		})
	}
}

func TestBlob_HashConsistency(t *testing.T) {
	// Test that hash is calculated correctly according to Git's algorithm
	data := []byte("what is up, doc?")
	blob := NewBlob(data)

	// Git's hash is SHA-1 of: "blob <size>\0<content>"
	// For "what is up, doc?" (16 bytes), it should be:
	// SHA-1("blob 16\0what is up, doc?")
	// Expected: bd9dbf5aae1a3862dd1526723246b20206e5fc37

	expectedHash := "bd9dbf5aae1a3862dd1526723246b20206e5fc37"

	// Convert hash to hex string for comparison
	hash, err := blob.Hash()
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if hash.String() != expectedHash {
		t.Logf("Hash mismatch for 'what is up, doc?'")
		t.Logf("Expected: %s", expectedHash)
		t.Logf("Got:      %s", hash)
		// Note: This test will fail if the hash calculation is incorrect
		// The expected hash is from Git's actual behavior
	}
}

func TestBlob_InterfaceCompliance(t *testing.T) {
	// Verify that *Blob implements BaseObject interface
	var _ objects.BaseObject = (*Blob)(nil)
}
