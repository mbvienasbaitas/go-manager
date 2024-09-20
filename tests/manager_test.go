package tests

import (
	"context"
	"errors"
	"github.com/mbvienasbaitas/go-manager"
	"github.com/mbvienasbaitas/go-manager/factory"
	"testing"
)

func TestManagerMake(t *testing.T) {
	ctx := context.Background()

	f, _ := factory.New[string](
		factory.OptionEvaluator[string](factory.FuncEvaluator[string](func(ctx context.Context, name string) bool {
			return name == "factory"
		})),
		factory.OptionBuilder[string](factory.FuncBuilder[string](func(ctx context.Context, name string) (string, error) {
			return "built", nil
		})),
	)

	m, _ := manager.New[string](
		manager.OptionFactory[string](f),
	)

	wantErr := manager.ErrFactoryNotSet

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
