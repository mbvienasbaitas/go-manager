package manager

import (
	"context"
	"errors"
	"testing"
)

func TestManagerMake(t *testing.T) {
	ctx := context.Background()

	m, _ := New[string](
		OptionFactory[string](FuncFactory[string](func(ctx context.Context, name string) (Service[string], error) {
			if name == "factory" {
				return NewGenericService[string]("built"), nil
			}

			return nil, ErrServiceUnsupported
		})),
	)

	wantErr := ErrFactoryNotSet

	_, err := m.Make(ctx, "fn")

	if !errors.Is(err, wantErr) {
		t.Errorf("want '%s', got '%s'", wantErr, err)
	}

	want := "built"

	result, _ := m.Make(ctx, "factory")

	if want != result {
		t.Errorf("want '%s', got '%s'", want, result)
	}
}
