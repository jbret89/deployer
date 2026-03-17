package service

import "strings"

const maxOutputLength = 8192

func sanitizeOutput(output string) string {
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.ReplaceAll(output, "\r", "\n")
	output = strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t':
			return r
		case r < 32 || r == 127:
			return -1
		default:
			return r
		}
	}, output)
	output = strings.TrimSpace(output)
	if len(output) > maxOutputLength {
		return output[:maxOutputLength] + "\n...[truncated]"
	}

	return output
}
