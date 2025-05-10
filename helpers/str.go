package helpers

import "strings"

func StrCoalesce(list ...string) string {
	for _, s := range list {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
