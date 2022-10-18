// generated code, do not modify
package mock

import (
	"context"
	example "example/codegen/example"
	mock "github.com/stretchr/testify/mock"
)

type MockExampleInterface struct {
	mock.Mock
}

func (m *MockExampleInterface) Method1(ctx context.Context, a string, x example.X) (example.Y, error) {
	args := m.Called(a, x)
	return args.Get(0).(example.Y), args.Error(1)
}

func (m *MockExampleInterface) Method2(ctx context.Context, a string, qp example.QueryParams) error {
	args := m.Called(a, qp)
	return args.Error(0)
}
