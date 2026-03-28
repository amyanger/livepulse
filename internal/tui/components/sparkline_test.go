package components

import "testing"

func TestSparkline_Render(t *testing.T) {
	s := NewSparkline(20)
	s.Push(10)
	s.Push(50)
	s.Push(100)
	s.Push(25)

	output := s.View()
	if len(output) == 0 {
		t.Error("expected non-empty sparkline output")
	}
}

func TestSparkline_MaxWidth(t *testing.T) {
	s := NewSparkline(5)
	for i := 0; i < 10; i++ {
		s.Push(float64(i * 10))
	}

	if len(s.values) != 5 {
		t.Errorf("expected 5 values, got %d", len(s.values))
	}
}
