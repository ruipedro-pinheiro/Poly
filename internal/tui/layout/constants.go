package layout

// Shared layout constants - single source of truth for all TUI dimensions.
// Every component should reference these instead of using magic numbers.
const (
	// Context window size in tokens (Anthropic default)
	DefaultContextWindow = 200_000

	// Sidebar
	SidebarWidth      = 28 // slightly wider for readability
	SidebarMinScreenW = 80 // screen must be wider than this to show sidebar

	// Terminal minimums
	MinTermWidth  = 60
	MinTermHeight = 15

	// Vertical zones (heights in rows)
	HeaderHeight    = 2
	StatusHeight    = 2 // top border + status line
	InputHeight     = 4 // editor box + hints line
	EditorMinHeight = 3
	EditorMaxHeight = 10

	// Dialogs
	DialogMaxWidth   = 80
	DialogMinWidth   = 40
	DialogWidthRatio = 0.8 // 80% of screen

	// Padding/borders
	ContentPadding   = 2 // left+right inside content area
	ChatAreaPadding  = 4 // total horizontal padding for chat viewport
	InputBoxPadding  = 8 // total horizontal padding for input textarea
	InputWidthOffset = 4 // total horizontal offset for input box border
	BorderWidth      = 1 // standard border thickness

	// Tool rendering
	ToolPreviewLines      = 3 // consistent tool result preview
	ToolErrorPreviewLines = 5 // more lines for error output

	// Splash
	SplashLogoWidth = 34 // width of the ASCII logo block
	SplashTitleLen  = 14 // visual width of "diamond P O L Y"

	// Content minimum
	ContentMinWidth = 30
)
