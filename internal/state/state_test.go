package state

import "testing"

type snap struct{ N int }

func TestFileRoundTrip(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	f, err := New[snap]("pr\x00o/r\x0042")
	if err != nil {
		t.Fatal(err)
	}
	if s, err := f.Load(); err != nil || s != nil {
		t.Fatalf("expected nothing saved yet, got %v, %v", s, err)
	}
	if err := f.Save(&snap{N: 7}); err != nil {
		t.Fatal(err)
	}
	s, err := f.Load()
	if err != nil || s == nil || s.N != 7 {
		t.Fatalf("expected N=7, got %v, %v", s, err)
	}

	other, _ := New[snap]("pr\x00o/r\x0043")
	if s, _ := other.Load(); s != nil {
		t.Errorf("expected a different watch to start fresh, got %v", s)
	}

	if err := f.Clear(); err != nil {
		t.Fatal(err)
	}
	if s, _ := f.Load(); s != nil {
		t.Errorf("expected cleared state, got %v", s)
	}
}
