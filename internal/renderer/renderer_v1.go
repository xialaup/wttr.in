package renderer

import (
	"github.com/chubin/wttr.in/internal/domain"
	w "github.com/chubin/wttr.in/internal/weather"
)

// Renderer Implementations (Stubs)
type V1Renderer struct{}

func (r *V1Renderer) Render(query domain.Query, localizer w.Localizer) (domain.RenderOutput, error) {
	// Stub: To be implemented
	return domain.RenderOutput{}, nil
}
