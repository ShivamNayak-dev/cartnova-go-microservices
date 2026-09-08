package events

import "testing"

func TestNewEventCreatesUniqueID(t *testing.T) {
	first, err := New("TestEvent", map[string]string{"value": "one"})
	if err != nil {
		t.Fatal(err)
	}

	second, err := New("TestEvent", map[string]string{"value": "two"})
	if err != nil {
		t.Fatal(err)
	}

	if first.ID == "" || second.ID == "" {
		t.Fatal("expected event IDs")
	}
	if first.ID == second.ID {
		t.Fatal("expected unique event IDs")
	}
}
