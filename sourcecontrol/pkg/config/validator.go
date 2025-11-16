package config

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Validator provides semantic validation for configuration values
type Validator struct{}

// ValidateKeyValue validates a configuration key-value pair
// Returns nil if valid, or an error describing the validation failure
func (v *Validator) ValidateKeyValue(key, value string) error {
	// Split key into parts
	parts := strings.Split(key, ".")
	if len(parts) < 2 {
		return NewInvalidValueError(key, fmt.Errorf("configuration key must have at least section.name format"))
	}

	section := parts[0]
	var subsection string
	var name string

	if len(parts) == 2 {
		name = parts[1]
	} else {
		subsection = strings.Join(parts[1:len(parts)-1], ".")
		name = parts[len(parts)-1]
	}

	// Validate based on section and name
	return v.validateBySection(section, subsection, name, value)
}

// validateBySection performs section-specific validation
func (v *Validator) validateBySection(section, subsection, name, value string) error {
	switch section {
	case "core":
		return v.validateCore(name, value)
	case "user":
		return v.validateUser(name, value)
	case "remote":
		return v.validateRemote(name, value)
	case "branch":
		return v.validateBranch(name, value)
	case "color":
		return v.validateColor(name, value)
	case "diff":
		return v.validateDiff(name, value)
	case "pull":
		return v.validatePull(name, value)
	case "push":
		return v.validatePush(name, value)
	case "init":
		return v.validateInit(name, value)
	default:
		// Unknown sections are allowed (extensibility)
		return nil
	}
}

// validateCore validates core.* configuration values
func (v *Validator) validateCore(name, value string) error {
	switch name {
	case "repositoryformatversion":
		return v.validateInt(value, "core.repositoryformatversion")
	case "filemode", "bare", "logallrefupdates", "ignorecase":
		return v.validateBoolean(value, "core."+name)
	case "autocrlf":
		return v.validateAutoCRLF(value)
	default:
		return nil
	}
}

// validateUser validates user.* configuration values
func (v *Validator) validateUser(name, value string) error {
	switch name {
	case "email":
		return v.validateEmail(value)
	case "name":
		if strings.TrimSpace(value) == "" {
			return NewInvalidValueError("user.name", fmt.Errorf("user name cannot be empty"))
		}
		return nil
	default:
		return nil
	}
}

// validateRemote validates remote.*.* configuration values
func (v *Validator) validateRemote(name, value string) error {
	switch name {
	case "url", "pushurl":
		return v.validateURL(value, "remote.*."+name)
	case "fetch", "push":
		return v.validateRefspec(value)
	default:
		return nil
	}
}

// validateBranch validates branch.*.* configuration values
func (v *Validator) validateBranch(name, value string) error {
	switch name {
	case "remote":
		if strings.TrimSpace(value) == "" {
			return NewInvalidValueError("branch.*.remote", fmt.Errorf("remote name cannot be empty"))
		}
		return nil
	case "merge":
		if !strings.HasPrefix(value, "refs/") {
			return NewInvalidValueError("branch.*.merge", fmt.Errorf("merge ref must start with 'refs/'"))
		}
		return nil
	case "rebase":
		return v.validateBoolean(value, "branch.*.rebase")
	default:
		return nil
	}
}

// validateColor validates color.* configuration values
func (v *Validator) validateColor(name, value string) error {
	switch name {
	case "ui":
		return v.validateColorUI(value)
	default:
		return nil
	}
}

// validateDiff validates diff.* configuration values
func (v *Validator) validateDiff(name, value string) error {
	switch name {
	case "renames":
		return v.validateBoolean(value, "diff.renames")
	default:
		return nil
	}
}

// validatePull validates pull.* configuration values
func (v *Validator) validatePull(name, value string) error {
	switch name {
	case "rebase":
		return v.validatePullRebase(value)
	default:
		return nil
	}
}

// validatePush validates push.* configuration values
func (v *Validator) validatePush(name, value string) error {
	switch name {
	case "default":
		return v.validatePushDefault(value)
	default:
		return nil
	}
}

// validateInit validates init.* configuration values
func (v *Validator) validateInit(name, value string) error {
	switch name {
	case "defaultbranch":
		return v.validateBranchName(value)
	default:
		return nil
	}
}

// Helper validation functions

func (v *Validator) validateInt(value, key string) error {
	if _, err := strconv.Atoi(value); err != nil {
		return NewInvalidValueError(key, fmt.Errorf("must be an integer: %v", err))
	}
	return nil
}

func (v *Validator) validateBoolean(value, key string) error {
	lower := strings.ToLower(strings.TrimSpace(value))
	validValues := []string{"true", "false", "yes", "no", "1", "0", "on", "off"}
	for _, valid := range validValues {
		if lower == valid {
			return nil
		}
	}
	return NewInvalidValueError(key, fmt.Errorf("must be a boolean (true/false/yes/no/1/0/on/off)"))
}

func (v *Validator) validateEmail(value string) error {
	if strings.TrimSpace(value) == "" {
		return NewInvalidValueError("user.email", fmt.Errorf("email cannot be empty"))
	}

	_, err := mail.ParseAddress(value)
	if err != nil {
		return NewInvalidValueError("user.email", fmt.Errorf("invalid email format: %v", err))
	}
	return nil
}

func (v *Validator) validateURL(value, key string) error {
	if strings.TrimSpace(value) == "" {
		return NewInvalidValueError(key, fmt.Errorf("URL cannot be empty"))
	}

	// Allow both absolute URLs and file paths
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, ".") || strings.Contains(value, ":\\") {
		// File path - basic validation
		return nil
	}

	// Parse as URL
	parsedURL, err := url.Parse(value)
	if err != nil {
		return NewInvalidValueError(key, fmt.Errorf("invalid URL: %v", err))
	}

	if parsedURL.Scheme == "" {
		return NewInvalidValueError(key, fmt.Errorf("URL must have a scheme (e.g., https://, git://, file://)"))
	}

	return nil
}

func (v *Validator) validateRefspec(value string) error {
	// Basic refspec validation
	// Format: [+]<src>:<dst> or [+]<src>
	if strings.TrimSpace(value) == "" {
		return NewInvalidValueError("refspec", fmt.Errorf("refspec cannot be empty"))
	}

	// Remove optional + prefix
	refspec := value
	if strings.HasPrefix(refspec, "+") {
		refspec = refspec[1:]
	}

	// Check for colon separator
	if strings.Contains(refspec, ":") {
		parts := strings.SplitN(refspec, ":", 2)
		if len(parts) != 2 {
			return NewInvalidValueError("refspec", fmt.Errorf("invalid refspec format"))
		}
		// Both parts should start with refs/ or contain wildcards
		for _, part := range parts {
			if part != "" && !strings.HasPrefix(part, "refs/") && !strings.Contains(part, "*") {
				return NewInvalidValueError("refspec", fmt.Errorf("refspec parts should start with 'refs/' or contain wildcards"))
			}
		}
	}

	return nil
}

func (v *Validator) validateAutoCRLF(value string) error {
	validValues := []string{"true", "false", "input"}
	lower := strings.ToLower(strings.TrimSpace(value))
	if slices.Contains(validValues, lower) {
		return nil
	}
	return NewInvalidValueError("core.autocrlf", fmt.Errorf("must be one of: true, false, input"))
}

func (v *Validator) validateColorUI(value string) error {
	validValues := []string{"auto", "always", "never", "true", "false"}
	lower := strings.ToLower(strings.TrimSpace(value))
	if slices.Contains(validValues, lower) {
		return nil
	}
	return NewInvalidValueError("color.ui", fmt.Errorf("must be one of: auto, always, never, true, false"))
}

func (v *Validator) validatePullRebase(value string) error {
	validValues := []string{"true", "false", "merges", "interactive"}
	lower := strings.ToLower(strings.TrimSpace(value))
	if slices.Contains(validValues, lower) {
		return nil
	}
	return NewInvalidValueError("pull.rebase", fmt.Errorf("must be one of: true, false, merges, interactive"))
}

func (v *Validator) validatePushDefault(value string) error {
	validValues := []string{"nothing", "current", "upstream", "simple", "matching"}
	lower := strings.ToLower(strings.TrimSpace(value))
	if slices.Contains(validValues, lower) {
		return nil
	}
	return NewInvalidValueError("push.default", fmt.Errorf("must be one of: nothing, current, upstream, simple, matching"))
}

func (v *Validator) validateBranchName(value string) error {
	if strings.TrimSpace(value) == "" {
		return NewInvalidValueError("branch.name", fmt.Errorf("branch name cannot be empty"))
	}

	// Git branch name restrictions
	invalidPatterns := []string{
		`^\.`,         // Cannot start with .
		`\.\.|@\{|\\`, // Cannot contain .., @{, or \
		`\.\.|//`,     // Cannot contain .. or //
		`[~^:?*\[\]]`, // Cannot contain ~, ^, :, ?, *, [, ]
		`\.lock$`,     // Cannot end with .lock
		`/$`,          // Cannot end with /
	}

	for _, pattern := range invalidPatterns {
		if matched, _ := regexp.MatchString(pattern, value); matched {
			return NewInvalidValueError("branch.name", fmt.Errorf("invalid branch name format"))
		}
	}

	return nil
}
