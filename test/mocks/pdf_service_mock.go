package mocks

import (
	"github.com/atoncetti/combiner/internal/service"
	"github.com/stretchr/testify/mock"
)

type PDFserviceMock struct {
	mock.Mock
}

func (m *PDFserviceMock) Generate(project *service.Project) ([]byte, error) {
	m.Called(project.GetFiles())

	return []byte("Ok, PDF returned"), nil
}
