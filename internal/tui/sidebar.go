package tui

import (
	"github.com/pedromelo/poly/internal/auth"
	"github.com/pedromelo/poly/internal/theme"
	"github.com/pedromelo/poly/internal/tui/components/sidebar"
	"github.com/pedromelo/poly/internal/tools"
	"image/color"
)

// renderSidebar renders the right info panel using the sidebar component
func (m Model) renderSidebar(width, height int) string {
	// Sync component state before rendering
	m.sidebarCmp.SetSize(width, height)
	m.sidebarCmp.SetProvider(m.defaultProvider, theme.ProviderColor(m.defaultProvider))
	m.sidebarCmp.SetThinkingMode(m.thinkingMode)
	m.sidebarCmp.SetTokenInfo(m.sessionInputTokens, m.sessionOutputTokens, m.sessionCost)
	m.sidebarCmp.SetYoloMode(tools.YoloMode)

	// Convert modified files
	modFiles := tools.GetModifiedFiles()
	sidebarFiles := make([]sidebar.ModifiedFile, len(modFiles))
	for i, f := range modFiles {
		sidebarFiles[i] = sidebar.ModifiedFile{Path: f}
	}
	m.sidebarCmp.SetModifiedFiles(sidebarFiles)

	// Provider statuses
	storage := auth.GetStorage()
	providers := []struct {
		name  string
		color color.Color
	}{
		{"claude", theme.Peach},
		{"gpt", theme.Green},
		{"gemini", theme.Blue},
		{"grok", theme.Sky},
	}
	statuses := make([]sidebar.ProviderStatus, len(providers))
	for i, p := range providers {
		statuses[i] = sidebar.ProviderStatus{
			Name:      p.name,
			Connected: storage.IsConnected(p.name),
			Color:     p.color,
		}
	}
	m.sidebarCmp.SetProviders(statuses)

	return m.sidebarCmp.View()
}
