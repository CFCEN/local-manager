package service

import "testing"

func TestAutomationHubRoutesAndCleansUp(t *testing.T) {
	h := NewAutomationHub()
	adapter := h.AddAdapter("a1")
	if err := h.Command("a1", map[string]string{"type": "action"}); err != nil {
		t.Fatal(err)
	}
	if got := string(<-adapter); got != `{"type":"action"}` {
		t.Fatalf("unexpected command %s", got)
	}

	browser := h.Subscribe("s1")
	h.Broadcast("s1", map[string]string{"type": "event"})
	if got := string(<-browser); got != `{"type":"event"}` {
		t.Fatalf("unexpected event %s", got)
	}

	h.RemoveAdapter("a1", adapter)
	if err := h.Command("a1", nil); err == nil {
		t.Fatal("expected offline error")
	}
	h.Unsubscribe("s1", browser)
}
