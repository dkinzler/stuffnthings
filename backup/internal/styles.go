package internal

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

type styles struct {
	ViewStyle             lipgloss.Style
	TitleStyle            lipgloss.Style
	NormalTextStyle       lipgloss.Style
	ErrorTextStyle        lipgloss.Style
	SelectedListItemStyle lipgloss.Style
	HelpStyles            help.Styles
}

func defaultStyles() styles {
	// copied these styles from the charmbracelet/bubbles/help package
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

	return styles{
		ViewStyle:             lipgloss.NewStyle().Margin(1),
		TitleStyle:            lipgloss.NewStyle().Bold(true).Padding(0, 3).Background(lipgloss.Color("#fc03ec")).Foreground(lipgloss.Color("#ffffff")),
		NormalTextStyle:       lipgloss.NewStyle(),
		ErrorTextStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")),
		SelectedListItemStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fc03ec")),
		HelpStyles:            helpStyles,
	}
}
