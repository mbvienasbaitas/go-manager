package manager

import (
	"errors"
	"testing"
	"time"
)

func TestTimedServiceGetService(t *testing.T) {
	svc := NewTimedService[string]("built", time.Now().Add(time.Second))

	result, err := svc.GetService()

	want := "built"

	if want != result {
		t.Errorf("want '%s', got '%s'", want, result)
	}

	if err != nil {
		t.Errorf("want err nil, got '%s'", err)
	}

	time.Sleep(time.Second)

	wantErr := ErrServiceExpired

	_, err = svc.GetService()

	if !errors.Is(err, wantErr) {
		t.Errorf("want '%s', got '%s'", wantErr, err)
	}
}
