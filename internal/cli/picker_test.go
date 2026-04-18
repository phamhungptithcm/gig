package cli

import (
	"bufio"
	"bytes"
	"testing"
)

func TestPickerModelFilterAndMove(t *testing.T) {
	model := newPickerModel([]pickerItem{
		{Value: "payments", Title: "payments", Subtitle: "github:acme/payments"},
		{Value: "billing", Title: "billing", Subtitle: "github:acme/billing", Recent: true},
		{Value: "frontend", Title: "frontend", Subtitle: "github:acme/frontend"},
	})

	model.move(1)
	if current, ok := model.current(); !ok || current.Value != "billing" {
		t.Fatalf("current after move = %#v, %v; want billing", current, ok)
	}

	model.appendFilter("pay")
	if current, ok := model.current(); !ok || current.Value != "payments" {
		t.Fatalf("current after filter = %#v, %v; want payments", current, ok)
	}
	if len(model.filtered) != 1 {
		t.Fatalf("len(model.filtered) = %d, want 1", len(model.filtered))
	}

	model.backspace()
	model.backspace()
	model.backspace()
	if len(model.filtered) != 3 {
		t.Fatalf("len(model.filtered) after clearing = %d, want 3", len(model.filtered))
	}
}

func TestReadPickerEvent(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want pickerEventKind
		text string
	}{
		{name: "arrow up", data: []byte{0x1b, '[', 'A'}, want: pickerEventUp},
		{name: "arrow down", data: []byte{0x1b, '[', 'B'}, want: pickerEventDown},
		{name: "space selects", data: []byte{' '}, want: pickerEventSelect},
		{name: "enter selects", data: []byte{'\n'}, want: pickerEventSelect},
		{name: "backspace", data: []byte{127}, want: pickerEventBackspace},
		{name: "text", data: []byte{'g'}, want: pickerEventText, text: "g"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event, err := readPickerEvent(bufio.NewReader(bytes.NewReader(test.data)))
			if err != nil {
				t.Fatalf("readPickerEvent() error = %v", err)
			}
			if event.Kind != test.want {
				t.Fatalf("event.Kind = %v, want %v", event.Kind, test.want)
			}
			if event.Text != test.text {
				t.Fatalf("event.Text = %q, want %q", event.Text, test.text)
			}
		})
	}
}
