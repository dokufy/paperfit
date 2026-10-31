package projectprovider

type RequestError struct {
	StatusCode int
	Message    string
	Err        error
}

func (r *RequestError) Error() string {
	if r == nil {
		return ""
	}
	if r.Err != nil {
		return r.Err.Error()
	}
	if r.Message != "" {
		return r.Message
	}
	return ""
}
