package style

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

type Styles struct {
	ViewStyle                lipgloss.Style
	TitleStyle               lipgloss.Style
	NormalTextStyle          lipgloss.Style
	ErrorTextStyle           lipgloss.Style
	ListItemTitleStyle       lipgloss.Style
	ListItemDescriptionStyle lipgloss.Style
	ListItemSelectedStyle    lipgloss.Style
	HelpStyles               help.Styles
}

func DefaultStyles() Styles {
	// copied from the charmbracelet/bubbles/help package
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#909090",
		Dark:  "#828282",
	})

	descStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#B2B2B2",
		Dark:  "#626262",
	})

	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#DDDADA",
		Dark:  "#3C3C3C",
	})
	helpStyles := help.Styles{
		Ellipsis:       sepStyle,
		ShortKey:       keyStyle,
		ShortDesc:      descStyle,
		ShortSeparator: sepStyle,
		FullKey:        keyStyle,
		FullDesc:       descStyle,
		FullSeparator:  sepStyle,
	}

	return Styles{
		ViewStyle:  lipgloss.NewStyle().Margin(1),
		TitleStyle: lipgloss.NewStyle().Bold(true).Padding(0, 3).Background(lipgloss.Color("#fc03ec")).Foreground(lipgloss.Color("#ffffff")),
		// Note: we could also set the max width of text dynamically using the current window size, but this is good enough for now
		NormalTextStyle:          lipgloss.NewStyle().Width(40),
		ErrorTextStyle:           lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Width(40),
		ListItemTitleStyle:       lipgloss.NewStyle(),
		ListItemDescriptionStyle: lipgloss.NewStyle().Faint(true),
		ListItemSelectedStyle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fc03ec")),
		HelpStyles:               helpStyles,
	}
}
