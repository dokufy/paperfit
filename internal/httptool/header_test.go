package httptool

import "testing"

func TestFlattenHeaders(t *testing.T) {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer test",
		"X-Trace-Id":    "trace",
	}

	const expected = "AuthorizationBearer testContent-Typeapplication/jsonX-Trace-Idtrace"

	for i := 0; i < 100; i++ {
		if got := FlattenHeaders(&headers); got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	}

	if got := FlattenHeaders(nil); got != "" {
		t.Fatalf("expected empty string for nil headers, got %q", got)
	}
}
