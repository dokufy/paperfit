package httptool

import (
	"sort"
	"strings"
)

func FlattenHeaders(headers *map[string]string) string {
	if headers == nil {
		return ""
	}

	keys := make([]string, 0, len(*headers))
	for key := range *headers {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	builder := strings.Builder{}

	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString((*headers)[key])
	}

	return builder.String()
}
