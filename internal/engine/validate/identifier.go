package validate

import (
	"fmt"
	"regexp"
	"strings"
)

var identifierRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// IsIdentifier reports whether s is a valid SQL identifier.
func IsIdentifier(s string) bool {
	return identifierRe.MatchString(s) && !isSQLKeyword(s)
}

// BucketName validates a bucket/table name.
func BucketName(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("bucket name is empty")
	}
	if len(s) > 128 {
		return fmt.Errorf("bucket name too long: %d", len(s))
	}
	if !IsIdentifier(s) {
		return fmt.Errorf("invalid bucket name: %q", s)
	}
	return nil
}

// PartitionName validates a partition suffix/name.
func PartitionName(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("partition name is empty")
	}
	if len(s) > 128 {
		return fmt.Errorf("partition name too long: %d", len(s))
	}
	if !IsIdentifier(s) {
		return fmt.Errorf("invalid partition name: %q", s)
	}
	return nil
}

// isSQLKeyword is a minimal guard against common SQL keywords.
func isSQLKeyword(s string) bool {
	switch strings.ToUpper(s) {
	case "SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TABLE",
		"WHERE", "FROM", "JOIN", "UNION", "ALL", "AND", "OR", "NOT", "NULL", "TRUE", "FALSE":
		return true
	}
	return false
}
