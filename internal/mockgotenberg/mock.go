package mockgotenberg

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	startOnce sync.Once
	started   bool
)

func EnsureAvailable(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	if host != "localhost" && host != "127.0.0.1" {
		return endpoint
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 300*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return endpoint
	}

	startOnce.Do(func() {
		ln, err := net.Listen("tcp", net.JoinHostPort(host, port))
		if err != nil {
			return
		}
		started = true
		srv := &http.Server{Handler: http.HandlerFunc(handle)}
		go func() {
			_ = srv.Serve(ln)
		}()
	})
	if !started {
		return endpoint
	}
	return endpoint
}

func handle(w http.ResponseWriter, r *http.Request) {
	text := extractText(r)
	pdf := buildPDF(text)
	w.Header().Set("Content-Type", "application/pdf")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

func extractText(r *http.Request) string {
	var b strings.Builder
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/") {
		mr, err := r.MultipartReader()
		if err == nil {
			for {
				part, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					break
				}
				data, _ := io.ReadAll(part)
				b.Write(data)
				b.WriteByte('\n')
			}
		}
	}
	_ = r.ParseMultipartForm(32 << 20)
	if r.MultipartForm != nil {
		for k, v := range r.MultipartForm.Value {
			for _, val := range v {
				b.WriteString(k)
				b.WriteByte('=')
				b.WriteString(val)
				b.WriteByte('\n')
			}
		}
		for _, files := range r.MultipartForm.File {
			for _, fh := range files {
				f, _ := fh.Open()
				if f != nil {
					data, _ := io.ReadAll(f)
					_ = f.Close()
					b.Write(data)
					b.WriteByte('\n')
				}
			}
		}
	}
	if b.Len() == 0 {
		data, _ := io.ReadAll(r.Body)
		b.Write(data)
	}
	return b.String()
}

func buildPDF(text string) []byte {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "(", "\\(")
	text = strings.ReplaceAll(text, ")", "\\)")

	lines := strings.Split(text, "\n")
	var content bytes.Buffer
	content.WriteString("BT\n/F1 9 Tf\n")
	y := 760
	for _, line := range lines {
		if y < 40 {
			break
		}
		if len(line) > 300 {
			line = line[:300]
		}
		// remove non-printable
		clean := strings.Map(func(r rune) rune {
			if r < 32 || r > 126 {
				return -1
			}
			return r
		}, line)
		fmt.Fprintf(&content, "1 0 0 1 40 %d Tm\n(%s) Tj\n", y, clean)
		y -= 11
	}
	content.WriteString("ET\n")

	stream := content.Bytes()
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := []int{}
	writeObj := func(n int, body string) {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", n, body)
	}
	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>")
	contentObj := fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), string(stream))
	writeObj(4, contentObj)
	writeObj(5, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")

	xrefPos := buf.Len()
	buf.WriteString("xref\n")
	fmt.Fprintf(&buf, "0 %d\n", len(offsets)+1)
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	buf.WriteString("trailer\n")
	fmt.Fprintf(&buf, "<< /Size %d /Root 1 0 R >>\n", len(offsets)+1)
	buf.WriteString("startxref\n")
	fmt.Fprintf(&buf, "%d\n", xrefPos)
	buf.WriteString("%%EOF\n")
	return buf.Bytes()
}
