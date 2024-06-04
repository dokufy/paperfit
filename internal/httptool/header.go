package httptool

func FlattenHeaders(headers map[string][]string) []byte {
	flattened := make([]byte, 0)

	for key, values := range headers {
		flattened = append(flattened, []byte(key)...)

		for _, value := range values {
			flattened = append(flattened, []byte(value)...)
		}
	}

	return flattened
}
