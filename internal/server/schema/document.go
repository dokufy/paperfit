package schema

import "mime/multipart"

type Input struct {
	Validate  bool
	Variables string
}

type DocumentInput struct {
	Body struct {
		ID    string
		Input Input
	}
}

type DocumentArchiveUrlInput struct {
	Body struct {
		URL     string
		Headers map[string][]string
		Input   Input
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
