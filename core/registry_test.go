package core

import (
	"context"
	"errors"
	"testing"
)

type stubComponent struct {
	BaseComponent
}

func (s *stubComponent) Mount(_ context.Context) error { return nil }
func (s *stubComponent) Render() (string, error)       { return "", nil }
func (s *stubComponent) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}
func (s *stubComponent) Unmount(_ context.Context) error { return nil }

func TestRegistry_RegisterAndCreate(t *testing.T) {
	reg := NewRegistry()

	err := reg.Register("stub", func() Component { return &stubComponent{} })
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	comp, err := reg.Create("stub")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, ok := comp.(*stubComponent); !ok {
		t.Fatalf("expected *stubComponent, got %T", comp)
	}
}

func TestRegistry_DuplicateRegister(t *testing.T) {
	reg := NewRegistry()

	factory := func() Component { return &stubComponent{} }

	if err := reg.Register("stub", factory); err != nil {
		t.Fatalf("first Register: %v", err)
	}

	err := reg.Register("stub", factory)
	if !errors.Is(err, ErrComponentAlreadyRegistered) {
		t.Fatalf("expected ErrComponentAlreadyRegistered, got %v", err)
	}
}

func TestRegistry_UnknownComponent(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Create("missing")
	if !errors.Is(err, ErrComponentNotRegistered) {
		t.Fatalf("expected ErrComponentNotRegistered, got %v", err)
	}
}

func TestRegistry_RegisterPageMode(t *testing.T) {
	reg := NewRegistry()

	if err := reg.RegisterPage("home", func() Component { return &stubComponent{} }, ModeSEO); err != nil {
		t.Fatal(err)
	}
	mode, ok := reg.Mode("home")
	if !ok || mode != ModeSEO {
		t.Fatalf("Mode = %v, %v; want ModeSEO", mode, ok)
	}

	if err := reg.Register("admin", func() Component { return &stubComponent{} }); err != nil {
		t.Fatal(err)
	}
	mode, ok = reg.Mode("admin")
	if !ok || mode != ModeLive {
		t.Fatalf("Mode = %v, %v; want ModeLive", mode, ok)
	}

	if _, ok := reg.Mode("missing"); ok {
		t.Fatal("expected missing mode ok=false")
	}
}
