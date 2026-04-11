package components

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

var blocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Sparkline renders a compact bar chart of recent values using Unicode block characters.
type Sparkline struct {
	values   []float64
	maxWidth int
}

// NewSparkline creates a sparkline that displays up to maxWidth data points.
func NewSparkline(maxWidth int) *Sparkline {
	return &Sparkline{maxWidth: maxWidth}
}

// Push adds a value to the sparkline, dropping the oldest if at capacity.
func (s *Sparkline) Push(val float64) {
	s.values = append(s.values, val)
	if len(s.values) > s.maxWidth {
		s.values = s.values[len(s.values)-s.maxWidth:]
	}
}

// View renders the sparkline as an uncolored string of Unicode block characters.
func (s *Sparkline) View() string {
	return s.render()
}

// ViewColored renders the sparkline with the given color applied.
func (s *Sparkline) ViewColored(c color.Color) string {
	raw := s.render()
	return lipgloss.NewStyle().Foreground(c).Render(raw)
}

// render builds the raw sparkline string.
func (s *Sparkline) render() string {
	if len(s.values) == 0 {
		return strings.Repeat(" ", s.maxWidth)
	}

	min, max := s.values[0], s.values[0]
	for _, v := range s.values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	var sb strings.Builder
	span := max - min
	for _, v := range s.values {
		idx := 0
		if span > 0 {
			idx = int((v - min) / span * float64(len(blocks)-1))
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		sb.WriteRune(blocks[idx])
	}

	for sb.Len() < s.maxWidth {
		sb.WriteRune(' ')
	}

	return sb.String()
}

// Clear removes all values from the sparkline.
func (s *Sparkline) Clear() {
	s.values = nil
}
