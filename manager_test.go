package manager

import (
	"context"
	"errors"
	"testing"
)

func TestManagerMake(t *testing.T) {
	ctx := context.Background()

	manager, _ := New[string]()

	wantErr := ErrFactoryNotSet

	_, err := manager.Make(ctx, "factory")

	if !errors.Is(err, wantErr) {
		t.Errorf("want '%s', got '%s'", wantErr, err)
	}

	manager.Options(OptionFactory[string]("factory", FuncFactory[string](func(ctx context.Context) (string, error) {
		return "built", nil
	})))

	want := "built"

	result, _ := manager.Make(ctx, "factory")

	if want != result {
		t.Errorf("want '%s', got '%s'", want, result)
	}
}
