package httptool

import "strings"

func FlattenHeaders(headers *map[string]string) string {
	if headers == nil {
		return ""
	}

	builder := strings.Builder{}

	for key, value := range *headers {
		builder.WriteString(key)
		builder.WriteString(value)
	}

	return builder.String()
}
