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

type DocumentArchiveInput struct {
	Body struct {
		URL     string
		Headers map[string][]string
		Input   Input
	}
}

type DocumentFileInput struct {
	RawBody multipart.Form
	Body    struct {
		Template []byte
		Input    Input
	}
}

type DocumentOutput struct {
	Body []byte
}
