package tag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/utkarsh5026/SourceControl/pkg/common/fileops"
	"github.com/utkarsh5026/SourceControl/pkg/config"
	"github.com/utkarsh5026/SourceControl/pkg/objects"
	"github.com/utkarsh5026/SourceControl/pkg/objects/commit"
	tagobj "github.com/utkarsh5026/SourceControl/pkg/objects/tag"
	"github.com/utkarsh5026/SourceControl/pkg/repository/refs"
	"github.com/utkarsh5026/SourceControl/pkg/repository/scpath"
	"github.com/utkarsh5026/SourceControl/pkg/repository/sourcerepo"
	"github.com/utkarsh5026/SourceControl/pkg/store"
)

// Manager manages Git tags in a repository
type Manager struct {
	repo       sourcerepo.Repository
	refManager *refs.RefManager
	store      *store.FileObjectStore
	config     *config.Manager
	tagsPath   scpath.SourcePath
}

// NewManager creates a new tag manager
func NewManager(repo sourcerepo.Repository) *Manager {
	sourceDir := repo.SourceDirectory()
	objStore := store.NewFileObjectStore()
	objStore.Initialize(repo.WorkingDirectory())
	return &Manager{
		repo:       repo,
		refManager: refs.NewRefManager(repo),
		store:      objStore,
		config:     config.NewManager(repo.WorkingDirectory()),
		tagsPath:   sourceDir.TagsPath(),
	}
}

// CreateTag creates a new tag
func (m *Manager) CreateTag(ctx context.Context, name string, objectRef string, opts ...CreateOption) error {
	options := &CreateOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Validate tag name
	if err := validateTagName(name); err != nil {
		return err
	}

	// Check if tag already exists
	if exists, _ := m.tagExists(name); exists && !options.Force {
		return NewErrTagExists(name)
	}

	// Resolve object reference (default to HEAD if empty)
	if objectRef == "" {
		objectRef = "HEAD"
	}

	objectSHA, err := m.resolveObject(objectRef)
	if err != nil {
		return NewErrInvalidObject(objectRef)
	}

	// Create tag based on type
	if options.Annotate || options.Sign {
		return m.createAnnotatedTag(name, objectSHA, options)
	}
	return m.createLightweightTag(name, objectSHA)
}

// createLightweightTag creates a lightweight tag
func (m *Manager) createLightweightTag(name string, objectSHA objects.ObjectHash) error {
	tagPath := m.getTagPath(name)
	tagContent := objectSHA.String() + "\n"

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(string(tagPath)), 0755); err != nil {
		return fmt.Errorf("failed to create tag directory: %w", err)
	}

	if err := fileops.WriteConfigString(tagPath, tagContent); err != nil {
		return fmt.Errorf("failed to write tag: %w", err)
	}

	return nil
}

// createAnnotatedTag creates an annotated tag
func (m *Manager) createAnnotatedTag(name string, objectSHA objects.ObjectHash, opts *CreateOptions) error {
	// Get tagger information from config
	tagger, err := m.getTaggerInfo(opts)
	if err != nil {
		return fmt.Errorf("failed to get tagger info: %w", err)
	}

	// Determine the object type
	objectType, err := m.getObjectType(objectSHA)
	if err != nil {
		return fmt.Errorf("failed to determine object type: %w", err)
	}

	// Build the tag object
	builder := tagobj.NewTagBuilder().
		ObjectHash(objectSHA).
		ObjectType(objectType).
		Name(name).
		Tagger(tagger)

	// Set message
	if opts.Message != "" {
		builder = builder.Message(opts.Message)
	} else {
		// Default message if none provided
		builder = builder.Message(fmt.Sprintf("Tag %s", name))
	}

	// Handle signed tags
	if opts.Sign {
		signature, err := m.signTagContent(builder, opts.LocalUser)
		if err != nil {
			return fmt.Errorf("failed to sign tag: %w", err)
		}
		builder = builder.Signature(signature)
	}

	// Build the tag
	tagObject, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build tag: %w", err)
	}

	// Write the tag object to the object store
	tagObjectSHA, err := m.store.WriteObject(tagObject)
	if err != nil {
		return fmt.Errorf("failed to write tag object: %w", err)
	}

	// Create the tag reference pointing to the tag object
	return m.createLightweightTag(name, tagObjectSHA)
}

// getTaggerInfo gets tagger information from options or config
func (m *Manager) getTaggerInfo(opts *CreateOptions) (*commit.CommitPerson, error) {
	var name, email string

	// Use options if provided
	if opts.TaggerName != "" {
		name = opts.TaggerName
	} else {
		// Fall back to config
		entry := m.config.Get("user.name")
		if entry != nil && entry.Value != "" {
			name = entry.Value
		} else {
			name = "Unknown User"
		}
	}

	if opts.TaggerEmail != "" {
		email = opts.TaggerEmail
	} else {
		// Fall back to config
		entry := m.config.Get("user.email")
		if entry != nil && entry.Value != "" {
			email = entry.Value
		} else {
			email = "unknown@example.com"
		}
	}

	return commit.NewCommitPerson(name, email, time.Now())
}

// getObjectType determines the type of an object by reading it
func (m *Manager) getObjectType(objectSHA objects.ObjectHash) (objects.ObjectType, error) {
	obj, err := m.store.ReadObject(objectSHA)
	if err != nil {
		return "", fmt.Errorf("failed to read object: %w", err)
	}
	return obj.Type(), nil
}

// signTagContent signs the tag content with GPG
func (m *Manager) signTagContent(builder *tagobj.TagBuilder, keyID string) (string, error) {
	// TODO: Implement GPG signing
	// For now, return an error indicating it's not implemented
	return "", fmt.Errorf("⚠️  GPG signing not yet implemented - please use annotated tags without signing for now")
}

// DeleteTag deletes a tag
func (m *Manager) DeleteTag(ctx context.Context, name string, opts ...DeleteOption) error {
	// Check if tag exists
	if exists, _ := m.tagExists(name); !exists {
		return NewErrTagNotFound(name)
	}

	tagPath := m.getTagPath(name)
	if err := os.Remove(string(tagPath)); err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	return nil
}

// ListTags lists all tags in the repository
func (m *Manager) ListTags(ctx context.Context, opts ...ListOption) ([]TagInfo, error) {
	options := &ListOptions{}
	for _, opt := range opts {
		opt(options)
	}

	var tags []TagInfo

	// Walk through tags directory
	err := filepath.Walk(string(m.tagsPath.ToAbsolutePath()), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get tag name relative to tags path
		relPath, err := filepath.Rel(string(m.tagsPath.ToAbsolutePath()), path)
		if err != nil {
			return err
		}

		// Convert path separators to forward slashes for consistency
		tagName := filepath.ToSlash(relPath)

		// Apply pattern filter if specified
		if options.Pattern != "" && !matchPattern(tagName, options.Pattern) {
			return nil
		}

		// Read tag content
		content, err := fileops.ReadStringStrict(scpath.AbsolutePath(path))
		if err != nil {
			return nil // Skip unreadable tags
		}

		content = strings.TrimSpace(content)
		sha, err := objects.NewObjectHashFromString(content)
		if err != nil {
			return nil // Skip invalid tags
		}

		tagInfo := TagInfo{
			Name: tagName,
			SHA:  sha,
			Type: Lightweight, // Default to lightweight
		}

		// Try to determine if it's an annotated tag by reading the object
		if obj, err := m.store.ReadObject(sha); err == nil && obj.Type() == objects.TagType {
			// It's a tag object, so it's either annotated or signed
			if tagObj, ok := obj.(*tagobj.Tag); ok {
				if tagObj.IsSigned() {
					tagInfo.Type = Signed
				} else {
					tagInfo.Type = Annotated
				}
				// Extract first line of message
				lines := strings.Split(tagObj.Message, "\n")
				if len(lines) > 0 {
					tagInfo.Message = strings.TrimSpace(lines[0])
				}
			} else {
				tagInfo.Type = Annotated
			}
		}

		tags = append(tags, tagInfo)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	// Sort tags
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})

	// Apply limit if specified
	if options.Limit > 0 && len(tags) > options.Limit {
		tags = tags[:options.Limit]
	}

	return tags, nil
}

// GetTag gets detailed information about a tag
func (m *Manager) GetTag(ctx context.Context, name string) (*Tag, error) {
	if exists, _ := m.tagExists(name); !exists {
		return nil, NewErrTagNotFound(name)
	}

	tagPath := m.getTagPath(name)
	content, err := fileops.ReadStringStrict(tagPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tag: %w", err)
	}

	content = strings.TrimSpace(content)
	sha, err := objects.NewObjectHashFromString(content)
	if err != nil {
		return nil, fmt.Errorf("invalid tag content: %w", err)
	}

	tag := &Tag{
		Name: name,
		SHA:  sha,
		Type: Lightweight,
	}

	// Check if it's an annotated tag
	if obj, err := m.store.ReadObject(sha); err == nil && obj.Type() == objects.TagType {
		// It's a tag object, parse it to get details
		if tagObj, ok := obj.(*tagobj.Tag); ok {
			if tagObj.IsSigned() {
				tag.Type = Signed
			} else {
				tag.Type = Annotated
			}
			tag.Message = tagObj.Message
			if tagObj.Tagger != nil {
				tag.Tagger = &Person{
					Name:  tagObj.Tagger.Name,
					Email: tagObj.Tagger.Email,
					When:  tagObj.Tagger.When.Time(),
				}
			}
			tag.ObjectType = string(tagObj.ObjectType)
		} else {
			tag.Type = Annotated
		}
	}

	return tag, nil
}

// tagExists checks if a tag exists
func (m *Manager) tagExists(name string) (bool, error) {
	tagPath := m.getTagPath(name)
	_, err := os.Stat(string(tagPath))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// getTagPath returns the filesystem path for a tag
func (m *Manager) getTagPath(name string) scpath.AbsolutePath {
	return scpath.AbsolutePath(filepath.Join(string(m.tagsPath.ToAbsolutePath()), name))
}

// resolveObject resolves an object reference to its SHA
func (m *Manager) resolveObject(ref string) (objects.ObjectHash, error) {
	// Try to resolve as a ref first
	if sha, err := m.refManager.ResolveToSHA(refs.RefPath(ref)); err == nil {
		return sha, nil
	}

	// Try to parse as a direct SHA
	if sha, err := objects.NewObjectHashFromString(ref); err == nil {
		return sha, nil
	}

	return "", fmt.Errorf("cannot resolve object: %s", ref)
}

// validateTagName validates a tag name according to Git rules
func validateTagName(name string) error {
	if name == "" {
		return NewErrInvalidTagName(name, "tag name cannot be empty")
	}

	// Tag name cannot start with a dot or hyphen
	if strings.HasPrefix(name, ".") {
		return NewErrInvalidTagName(name, "cannot start with a dot")
	}

	if strings.HasPrefix(name, "-") {
		return NewErrInvalidTagName(name, "cannot start with a hyphen")
	}

	// Tag name cannot end with .lock
	if strings.HasSuffix(name, ".lock") {
		return NewErrInvalidTagName(name, "cannot end with .lock")
	}

	// Tag name cannot contain certain characters
	invalidChars := []string{"~", "^", ":", "?", "*", "[", "\\", " ", "\t", "\n"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return NewErrInvalidTagName(name, fmt.Sprintf("cannot contain '%s'", char))
		}
	}

	// Tag name cannot contain consecutive dots
	if strings.Contains(name, "..") {
		return NewErrInvalidTagName(name, "cannot contain consecutive dots")
	}

	// Tag name cannot contain @{
	if strings.Contains(name, "@{") {
		return NewErrInvalidTagName(name, "cannot contain '@{'")
	}

	return nil
}

// matchPattern matches a tag name against a pattern
func matchPattern(name, pattern string) bool {
	// Simple glob pattern matching (supports * wildcard)
	// TODO: Implement more sophisticated pattern matching
	if pattern == "*" {
		return true
	}

	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(name, prefix)
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(name, suffix)
	}

	return name == pattern
}
