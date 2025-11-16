package commit

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	if len(s) == 0 {
		return false
	}

	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// looksLikeCommitSHA returns true if the string looks like it could be a commit SHA
// This includes both full (40 char) and short (7+ char) SHAs
func LooksLikeCommitSHA(s string) bool {
	if !isHexString(s) {
		return false
	}
	length := len(s)
	return length >= 4 && length <= 40
}
