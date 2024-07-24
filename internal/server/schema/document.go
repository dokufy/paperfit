package schema

import "mime/multipart"

type Input struct {
	Validate  bool
	Variables string
}

type DocumentInput struct {
	Body struct {
		ID        string
		Validate  bool
		Variables string
	}
}

type DocumentArchiveUrlInput struct {
	Body struct {
		URL       string
		Headers   map[string][]string
		Validate  bool
		Variables string
	}
}

type DocumentArchiveInput struct {
	RawBody   multipart.Form
	Validate  bool
	Variables string
}

type DocumentFileInput struct {
	RawBody   multipart.Form
	Validate  bool
	Variables string
}

type DocumentOutput struct {
	Body []byte
}
