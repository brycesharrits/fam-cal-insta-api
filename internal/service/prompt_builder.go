package service

import "fmt"

var monthNames = [13]string{
	"", "January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

// ThemePromptConfig holds the base prompt template and style descriptors for a theme.
type ThemePromptConfig struct {
	StyleDescriptor string // injected into every month's prompt
	MoodPrefix      string // opening tone/mood words
}

var themeConfigs = map[string]ThemePromptConfig{
	"goofy_holiday": {
		StyleDescriptor: "whimsical illustrated holiday scene, playful and funny, warm festive colors, cartoonish art style",
		MoodPrefix:      "Humorous and joyful",
	},
	"watercolor": {
		StyleDescriptor: "soft watercolor painting, delicate brushstrokes, pastel palette, artistic and dreamy",
		MoodPrefix:      "Gentle and artistic",
	},
	"vintage_film": {
		StyleDescriptor: "vintage film photography aesthetic, warm grain, faded tones, nostalgic 1970s family album style",
		MoodPrefix:      "Nostalgic and warm",
	},
	"modern_minimal": {
		StyleDescriptor: "clean modern illustration, simple geometric shapes, bold accent colors on white, Scandinavian design influence",
		MoodPrefix:      "Clean and contemporary",
	},
	"cozy_illustrated": {
		StyleDescriptor: "cozy illustrated scene, warm earthy tones, hand-drawn style, heartwarming and intimate",
		MoodPrefix:      "Warm and cozy",
	},
	"nature_botanical": {
		StyleDescriptor: "botanical illustration style, lush natural elements, seasonal flora and fauna, elegant line art with color washes",
		MoodPrefix:      "Natural and serene",
	},
}

// PromptBuilder constructs image generation prompts from theme + month + reference context.
type PromptBuilder struct{}

func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// Build constructs a Flux-ready prompt for a calendar month image.
// referenceContext is a brief description derived from analyzing the reference photo
// (e.g., "a snowy outdoor scene with children playing").
func (p *PromptBuilder) Build(theme string, month int, year int, referenceContext string) string {
	monthName := monthNames[month]
	cfg, ok := themeConfigs[theme]
	if !ok {
		// Fallback for unknown themes — use the theme string directly
		cfg = ThemePromptConfig{
			StyleDescriptor: theme,
			MoodPrefix:      "Beautiful and memorable",
		}
	}

	prompt := fmt.Sprintf(
		"%s family calendar image for %s %d. %s. "+
			"The scene should evoke the season and spirit of %s, inspired by: %s. "+
			"Calendar art, high quality, suitable for printing, 2400x1800 pixels, landscape orientation. "+
			"Style: %s.",
		cfg.MoodPrefix,
		monthName,
		year,
		seasonDescriptor(month),
		monthName,
		referenceContext,
		cfg.StyleDescriptor,
	)

	return prompt
}

func seasonDescriptor(month int) string {
	switch {
	case month == 12 || month <= 2:
		return "winter season, cold, cozy indoors, holiday atmosphere"
	case month <= 5:
		return "spring season, renewal, blooming flowers, fresh beginnings"
	case month <= 8:
		return "summer season, warmth, outdoor activities, sunshine and blue skies"
	default:
		return "autumn season, falling leaves, harvest colors, cozy sweater weather"
	}
}
