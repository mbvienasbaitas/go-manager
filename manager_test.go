package manager

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

// barrierExpiredService reports expiry, but only once `wait` callers have asked.
type barrierExpiredService struct {
	seen int32
	wait int32
}

func (receiver *barrierExpiredService) GetService() (int, error) {
	atomic.AddInt32(&receiver.seen, 1)

	deadline := time.Now().Add(time.Second)

	for atomic.LoadInt32(&receiver.seen) < receiver.wait && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}

	return 0, ErrServiceExpired
}

func TestManagerMakeConcurrentExpiredBuildsOnce(t *testing.T) {
	ctx := context.Background()

	const callers = 20

	var builds int32

	m, _ := New[int](
		OptionFactory[int](FuncFactory[int](func(ctx context.Context, name string) (Service[int], error) {
			n := atomic.AddInt32(&builds, 1)

			time.Sleep(20 * time.Millisecond)

			return NewTimedService[int](int(n), time.Now().Add(time.Hour)), nil
		})),
	)

	// Seed an expired service that holds every caller until all of them have
	// observed the expiry, so they all race into the rebuild path together.
	m.registry["svc"] = &barrierExpiredService{wait: callers}

	var wg sync.WaitGroup

	for i := 0; i < callers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if _, err := m.Make(ctx, "svc"); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		}()
	}

	wg.Wait()

	if got := atomic.LoadInt32(&builds); got != 1 {
		t.Errorf("want 1 build, got %d", got)
	}
}

type failingService struct{}

func (failingService) GetService() (string, error) {
	return "", errors.New("boom")
}

func TestManagerMakeReleasesLockOnServiceError(t *testing.T) {
	m, _ := New[string]()

	m.registry["svc"] = failingService{}

	if _, err := m.Make(context.Background(), "svc"); err == nil {
		t.Fatal("want error, got nil")
	}

	done := make(chan struct{})

	go func() {
		_ = m.Forget("svc")

		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("read lock was not released")
	}
}

// trackedService is a test service whose expiry, boot and shutdown are controllable.
type trackedService struct {
	value       int
	expired     atomic.Bool
	bootErr     error
	shutdownErr error
	shutdowns   atomic.Int32
}

func (receiver *trackedService) GetService() (int, error) {
	if receiver.expired.Load() {
		return receiver.value, ErrServiceExpired
	}

	return receiver.value, nil
}

func (receiver *trackedService) Boot() error {
	return receiver.bootErr
}

func (receiver *trackedService) Shutdown() error {
	receiver.shutdowns.Add(1)

	return receiver.shutdownErr
}

func TestManagerMakeRebuildsExpired(t *testing.T) {
	var (
		mu    sync.Mutex
		built []*trackedService
	)

	m, _ := New[int](
		OptionFactory[int](FuncFactory[int](func(ctx context.Context, name string) (Service[int], error) {
			mu.Lock()
			defer mu.Unlock()

			svc := &trackedService{value: len(built) + 1}

			built = append(built, svc)

			return svc, nil
		})),
	)

	ctx := context.Background()

	first, err := m.Make(ctx, "svc")

	if err != nil || first != 1 {
		t.Fatalf("first Make: want (1, nil), got (%d, %v)", first, err)
	}

	again, err := m.Make(ctx, "svc")

	if err != nil || again != 1 {
		t.Fatalf("cached Make: want (1, nil), got (%d, %v)", again, err)
	}

	built[0].expired.Store(true)

	second, err := m.Make(ctx, "svc")

	if err != nil || second != 2 {
		t.Fatalf("Make after expiry: want (2, nil), got (%d, %v)", second, err)
	}

	if got := built[0].shutdowns.Load(); got != 1 {
		t.Errorf("want expired service shut down once, got %d", got)
	}
}

func TestManagerMakeReplacesExpiredEvenIfShutdownFails(t *testing.T) {
	shutdownErr := errors.New("conn already closed")

	var reported []error

	var builds int32

	m, _ := New[int](
		OptionFactory[int](FuncFactory[int](func(ctx context.Context, name string) (Service[int], error) {
			return &trackedService{value: int(atomic.AddInt32(&builds, 1))}, nil
		})),
		OptionErrorHandler[int](func(name string, err error) {
			reported = append(reported, err)
		}),
	)

	stale := &trackedService{shutdownErr: shutdownErr}

	stale.expired.Store(true)

	m.registry["svc"] = stale

	for i := 0; i < 2; i++ {
		got, err := m.Make(context.Background(), "svc")

		if err != nil || got != 1 {
			t.Fatalf("Make #%d: want (1, nil), got (%d, %v)", i+1, got, err)
		}
	}

	if len(reported) != 1 || !errors.Is(reported[0], shutdownErr) {
		t.Errorf("want shutdown error reported once, got %v", reported)
	}
}

func TestManagerMakeShutsDownServiceThatFailsToBoot(t *testing.T) {
	bootErr := errors.New("ping failed")

	svc := &trackedService{bootErr: bootErr}

	m, _ := New[int](
		OptionFactory[int](FuncFactory[int](func(ctx context.Context, name string) (Service[int], error) {
			return svc, nil
		})),
	)

	if _, err := m.Make(context.Background(), "svc"); !errors.Is(err, bootErr) {
		t.Fatalf("want boot error, got %v", err)
	}

	if got := svc.shutdowns.Load(); got != 1 {
		t.Errorf("want failed service shut down once, got %d", got)
	}

	if _, ok := m.registry["svc"]; ok {
		t.Error("service that failed to boot was registered")
	}
}

func TestManagerShutdownStopsAllAndResets(t *testing.T) {
	var builds int32

	m, _ := New[int](
		OptionFactory[int](FuncFactory[int](func(ctx context.Context, name string) (Service[int], error) {
			return &trackedService{value: int(atomic.AddInt32(&builds, 1))}, nil
		})),
	)

	failErr := errors.New("boom")

	services := map[string]*trackedService{
		"a": {value: 1},
		"b": {value: 2, shutdownErr: failErr},
		"c": {value: 3},
	}

	for name, svc := range services {
		m.registry[name] = svc
	}

	if err := m.Shutdown(); !errors.Is(err, failErr) {
		t.Errorf("want shutdown error, got %v", err)
	}

	for name, svc := range services {
		if got := svc.shutdowns.Load(); got != 1 {
			t.Errorf("service %s: want 1 shutdown, got %d", name, got)
		}
	}

	if _, err := m.Make(context.Background(), "a"); err != nil {
		t.Fatalf("Make after Shutdown: %v", err)
	}

	if got := atomic.LoadInt32(&builds); got != 1 {
		t.Errorf("want Make after Shutdown to build a new service, got %d builds", got)
	}
}

// blockingFactory builds a trackedService for "svc", signalling on started and
// then waiting for release before returning it.
func blockingFactory(started chan<- struct{}, release <-chan struct{}, built **trackedService) Option[int] {
	return OptionFactory[int](FuncFactory[int](func(ctx context.Context, name string) (Service[int], error) {
		svc := &trackedService{value: 1}

		*built = svc

		close(started)

		<-release

		return svc, nil
	}))
}

func TestManagerShutdownWaitsForBuild(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})

	var built *trackedService

	m, _ := New[int](blockingFactory(started, release, &built))

	makeDone := make(chan error, 1)

	go func() {
		_, err := m.Make(context.Background(), "svc")

		makeDone <- err
	}()

	<-started

	shutdownDone := make(chan error, 1)

	go func() {
		shutdownDone <- m.Shutdown()
	}()

	select {
	case <-shutdownDone:
		t.Fatal("Shutdown returned before the in-flight build finished")
	case <-time.After(20 * time.Millisecond):
	}

	close(release)

	if err := <-makeDone; err != nil {
		t.Fatalf("Make: %v", err)
	}

	if err := <-shutdownDone; err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if got := built.shutdowns.Load(); got != 1 {
		t.Errorf("want service built during Shutdown to be shut down once, got %d", got)
	}

	if _, ok := m.registry["svc"]; ok {
		t.Error("service is still registered after Shutdown")
	}
}

func TestManagerForgetDuringBuildDoesNotLeak(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})

	var built *trackedService

	m, _ := New[int](blockingFactory(started, release, &built))

	makeDone := make(chan error, 1)

	go func() {
		_, err := m.Make(context.Background(), "svc")

		makeDone <- err
	}()

	<-started

	forgetDone := make(chan error, 1)

	go func() {
		forgetDone <- m.Forget("svc")
	}()

	select {
	case <-forgetDone:
		t.Fatal("Forget returned before the in-flight build finished")
	case <-time.After(20 * time.Millisecond):
	}

	close(release)

	if err := <-makeDone; err != nil {
		t.Fatalf("Make: %v", err)
	}

	if err := <-forgetDone; err != nil {
		t.Fatalf("Forget: %v", err)
	}

	if got := built.shutdowns.Load(); got != 1 {
		t.Errorf("want forgotten service shut down once, got %d", got)
	}

	if _, ok := m.registry["svc"]; ok {
		t.Error("service is still registered after Forget")
	}
}

func TestManagerErrorHandlerMayCallMake(t *testing.T) {
	var m *Manager[int]

	m, _ = New[int](
		OptionFactory[int](FuncFactory[int](func(ctx context.Context, name string) (Service[int], error) {
			return &trackedService{value: 1}, nil
		})),
		OptionErrorHandler[int](func(name string, err error) {
			_, _ = m.Make(context.Background(), name)
		}),
	)

	stale := &trackedService{shutdownErr: errors.New("conn already closed")}

	stale.expired.Store(true)

	m.registry["svc"] = stale

	done := make(chan struct{})

	go func() {
		_, _ = m.Make(context.Background(), "svc")

		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("error handler calling Make deadlocked")
	}
}

func TestManagerMakeRejectsServiceExpiredOnBuild(t *testing.T) {
	svc := &trackedService{}

	svc.expired.Store(true)

	m, _ := New[int](
		OptionFactory[int](FuncFactory[int](func(ctx context.Context, name string) (Service[int], error) {
			return svc, nil
		})),
	)

	if _, err := m.Make(context.Background(), "svc"); !errors.Is(err, ErrServiceExpired) {
		t.Fatalf("want ErrServiceExpired, got %v", err)
	}

	if got := svc.shutdowns.Load(); got != 1 {
		t.Errorf("want unusable service shut down once, got %d", got)
	}

	if _, ok := m.registry["svc"]; ok {
		t.Error("service expired on build was registered")
	}
}
