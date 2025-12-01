package tui

import (
	"fmt"
	"strings"

	"github.com/Gylmynnn/goclean/internal/cleaner"
	"github.com/Gylmynnn/goclean/internal/scanner"
	"github.com/Gylmynnn/goclean/internal/tui/components"
	"github.com/Gylmynnn/goclean/internal/tui/styles"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View states
type viewState int

const (
	viewMain viewState = iota
	viewDetails
	viewConfirm
	viewPassword
	viewCleaning
	viewResults
)

// Messages
type scanCompleteMsg struct {
	results []cleaner.ScanResult
}

type cleanCompleteMsg struct {
	results []cleaner.CleanResult
}

type errMsg struct {
	err error
}

type passwordValidMsg struct {
	valid bool
	err   error
}

// CategoryData holds scan data for a category
type CategoryData struct {
	Info       cleaner.CategoryInfo
	ScanResult cleaner.ScanResult
	Selected   bool
}

// Model is the main TUI model
type Model struct {
	state  viewState
	width  int
	height int

	categories   []CategoryData
	scanResults  []cleaner.ScanResult
	cleanResults []cleaner.CleanResult

	cursor         int
	detailCursor   int
	confirmCursor  int
	activeCategory int

	scrollOffset       int
	detailScrollOffset int

	spinner spinner.Model

	scanner *scanner.Scanner
	cleaner *cleaner.Cleaner

	isScanning    bool
	isCleaning    bool
	errorMessage  string
	password      string
	passwordError string
	isValidating  bool
}

func New() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	return Model{
		state:   viewMain,
		spinner: s,
		scanner: scanner.New(),
		cleaner: cleaner.New(cleaner.CleanOptions{KeepLastN: 1}),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.scanCategories)
}

func (m Model) scanCategories() tea.Msg {
	return scanCompleteMsg{results: m.scanner.ScanAll()}
}

func (m Model) cleanSelected() tea.Msg {
	var items []cleaner.CleanableItem
	for _, cat := range m.categories {
		// Include items from selected categories
		if cat.Selected {
			for i := range cat.ScanResult.Items {
				cat.ScanResult.Items[i].IsSelected = true
				items = append(items, cat.ScanResult.Items[i])
			}
		} else {
			// Also include individually selected items
			for _, item := range cat.ScanResult.Items {
				if item.IsSelected {
					items = append(items, item)
				}
			}
		}
	}
	return cleanCompleteMsg{results: m.cleaner.Clean(items)}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case scanCompleteMsg:
		m.isScanning = false
		m.scanResults = msg.results
		m.initCategories()
		return m, nil
	case cleanCompleteMsg:
		m.isCleaning = false
		m.cleanResults = msg.results
		m.state = viewResults
		return m, nil
	case passwordValidMsg:
		m.isValidating = false
		if msg.valid {
			// Password is correct, proceed with cleaning
			m.isCleaning = true
			m.state = viewCleaning
			return m, m.cleanSelected
		}
		// Password is incorrect
		m.passwordError = "Incorrect password"
		m.password = ""
		return m, nil
	case errMsg:
		m.errorMessage = msg.err.Error()
		return m, nil
	}
	return m, nil
}

func (m *Model) initCategories() {
	categoryInfos := cleaner.GetCategoryInfo()
	m.categories = make([]CategoryData, 0, len(categoryInfos))
	for _, info := range categoryInfos {
		var scanResult cleaner.ScanResult
		for _, result := range m.scanResults {
			if result.Category == info.Category {
				scanResult = result
				break
			}
		}
		m.categories = append(m.categories, CategoryData{
			Info:       info,
			ScanResult: scanResult,
			Selected:   false,
		})
	}
}

func (m Model) visibleItems() int {
	// Use full height, reserve space for header and summary only
	available := m.height - 8
	if available < 3 {
		return 3
	}
	return available
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case viewMain:
		return m.handleMainKeys(msg)
	case viewDetails:
		return m.handleDetailKeys(msg)
	case viewConfirm:
		return m.handleConfirmKeys(msg)
	case viewPassword:
		return m.handlePasswordKeys(msg)
	case viewResults:
		return m.handleResultKeys(msg)
	}
	return m, nil
}

func (m Model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visible := m.visibleItems()
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.scrollOffset {
				m.scrollOffset = m.cursor
			}
		}
	case "down", "j":
		if m.cursor < len(m.categories)-1 {
			m.cursor++
			if m.cursor >= m.scrollOffset+visible {
				m.scrollOffset = m.cursor - visible + 1
			}
		}
	case " ":
		if m.cursor < len(m.categories) {
			m.categories[m.cursor].Selected = !m.categories[m.cursor].Selected
		}
	case "enter":
		if m.cursor < len(m.categories) {
			m.activeCategory = m.cursor
			m.detailCursor = 0
			m.detailScrollOffset = 0
			m.state = viewDetails
		}
	case "a":
		for i := range m.categories {
			m.categories[i].Selected = true
		}
	case "n":
		for i := range m.categories {
			m.categories[i].Selected = false
		}
	case "c":
		if m.hasSelectedCategories() {
			m.confirmCursor = 0
			m.state = viewConfirm
		}
	case "r":
		m.isScanning = true
		return m, m.scanCategories
	}
	return m, nil
}

func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := m.categories[m.activeCategory].ScanResult.Items
	visible := m.visibleItems()
	maxCursor := len(items) - 1

	switch msg.String() {
	case "q", "esc":
		m.state = viewMain
	case "up", "k":
		if m.detailCursor > 0 {
			m.detailCursor--
			if m.detailCursor < m.detailScrollOffset {
				m.detailScrollOffset = m.detailCursor
			}
		}
	case "down", "j":
		if m.detailCursor < maxCursor {
			m.detailCursor++
			if m.detailCursor >= m.detailScrollOffset+visible {
				m.detailScrollOffset = m.detailCursor - visible + 1
			}
		}
	case " ":
		itemsPtr := &m.categories[m.activeCategory].ScanResult.Items
		if m.detailCursor < len(*itemsPtr) {
			(*itemsPtr)[m.detailCursor].IsSelected = !(*itemsPtr)[m.detailCursor].IsSelected
		}
	case "a":
		// Select all items in this category
		itemsPtr := &m.categories[m.activeCategory].ScanResult.Items
		for i := range *itemsPtr {
			(*itemsPtr)[i].IsSelected = true
		}
	case "n":
		// Deselect all items in this category
		itemsPtr := &m.categories[m.activeCategory].ScanResult.Items
		for i := range *itemsPtr {
			(*itemsPtr)[i].IsSelected = false
		}
	case "c":
		// Clean selected items in this category
		if m.hasSelectedItemsInCategory() {
			m.confirmCursor = 0
			m.state = viewConfirm
		}
	}
	return m, nil
}

func (m Model) hasSelectedItemsInCategory() bool {
	items := m.categories[m.activeCategory].ScanResult.Items
	for _, item := range items {
		if item.IsSelected {
			return true
		}
	}
	return false
}

func (m Model) handleConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.state = viewMain
	case "left", "h":
		m.confirmCursor = 0
	case "right", "l":
		m.confirmCursor = 1
	case "enter":
		if m.confirmCursor == 1 {
			return m.startCleaning()
		}
		m.state = viewMain
	case "y":
		return m.startCleaning()
	}
	return m, nil
}

func (m Model) startCleaning() (tea.Model, tea.Cmd) {
	// Check if we need root access
	selectedCategories := m.getSelectedCategories()
	if cleaner.NeedsRoot(selectedCategories) {
		m.password = ""
		m.passwordError = ""
		m.state = viewPassword
		return m, nil
	}

	// No root needed, proceed with cleaning
	m.isCleaning = true
	m.state = viewCleaning
	return m, m.cleanSelected
}

func (m Model) getSelectedCategories() []cleaner.CleanCategory {
	var categories []cleaner.CleanCategory
	for _, cat := range m.categories {
		if cat.Selected {
			categories = append(categories, cat.Info.Category)
		}
	}
	// Also check for individually selected items
	for _, cat := range m.categories {
		hasSelected := false
		for _, item := range cat.ScanResult.Items {
			if item.IsSelected {
				hasSelected = true
				break
			}
		}
		if hasSelected && !cat.Selected {
			categories = append(categories, cat.Info.Category)
		}
	}
	return categories
}

func (m Model) handlePasswordKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = viewConfirm
		m.password = ""
		m.passwordError = ""
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		if len(m.password) > 0 {
			m.isValidating = true
			m.passwordError = ""
			return m, m.validatePassword
		}
	case "backspace":
		if len(m.password) > 0 {
			m.password = m.password[:len(m.password)-1]
		}
	default:
		// Only add printable characters
		if len(msg.String()) == 1 && msg.String()[0] >= 32 && msg.String()[0] <= 126 {
			m.password += msg.String()
		}
	}
	return m, nil
}

func (m Model) validatePassword() tea.Msg {
	m.cleaner.SetPassword(m.password)
	err := m.cleaner.ValidatePassword()
	return passwordValidMsg{valid: err == nil, err: err}
}

func (m Model) handleResultKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "r", "enter":
		m.isScanning = true
		m.state = viewMain
		return m, m.scanCategories
	}
	return m, nil
}

func (m Model) hasSelectedCategories() bool {
	for _, cat := range m.categories {
		if cat.Selected {
			return true
		}
	}
	return false
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string
	switch m.state {
	case viewMain:
		content = m.renderMainView()
	case viewDetails:
		content = m.renderDetailView()
	case viewConfirm:
		content = m.renderConfirmView()
	case viewPassword:
		content = m.renderPasswordView()
	case viewCleaning:
		content = m.renderCleaningView()
	case viewResults:
		content = m.renderResultsView()
	}

	// Center content both horizontally and vertically
	contentWidth := styles.GetContentWidth(m.width)
	return styles.CenterBoth(content, m.width, m.height, contentWidth)
}

// ============ RENDER VIEWS ============

func (m Model) renderMainView() string {
	if styles.IsDesktop(m.width) {
		return m.renderMainDesktop()
	}
	return m.renderMainMobile()
}

func (m Model) renderMainMobile() string {
	var b strings.Builder

	// Compact header
	b.WriteString(styles.TitleStyle.Render(" GoClean"))
	b.WriteString("\n\n")

	if m.isScanning {
		b.WriteString(fmt.Sprintf(" %s Scanning...\n", m.spinner.View()))
		return b.String()
	}

	// Categories
	visible := m.visibleItems()
	endIdx := min(m.scrollOffset+visible, len(m.categories))

	for i := m.scrollOffset; i < endIdx; i++ {
		cat := m.categories[i]
		b.WriteString(m.renderCategoryItemMobile(cat, i == m.cursor))
		b.WriteString("\n")
	}

	// Scroll hint
	if endIdx < len(m.categories) {
		b.WriteString(styles.TextMutedStyle.Render(" ↓ more\n"))
	}

	// Summary
	b.WriteString("\n")
	totalSize, count := m.calculateSelected()
	if count > 0 {
		b.WriteString(styles.SuccessStyle.Render(fmt.Sprintf(" %d selected", count)))
		b.WriteString(styles.TextMutedStyle.Render(fmt.Sprintf(" • %s", components.FormatSize(totalSize))))
	} else {
		b.WriteString(styles.TextMutedStyle.Render(" No items selected"))
	}

	return b.String()
}

func (m Model) renderMainDesktop() string {
	var b strings.Builder
	contentWidth := styles.GetContentWidth(m.width)

	// Header with border
	header := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.SurfaceAlt).
		Padding(0, 2).
		Width(contentWidth).
		Render(styles.TitleStyle.Render("GoClean") + "  " + styles.SubtitleStyle.Render("Arch Linux System Cleaner"))

	b.WriteString(header)
	b.WriteString("\n\n")

	if m.isScanning {
		b.WriteString(fmt.Sprintf("  %s Scanning system...\n", m.spinner.View()))
		return b.String()
	}

	// Categories section
	b.WriteString(styles.HeaderStyle.Render("  Categories"))
	if len(m.categories) > m.visibleItems() {
		b.WriteString(styles.TextMutedStyle.Render(fmt.Sprintf("  [%d/%d]", m.cursor+1, len(m.categories))))
	}
	b.WriteString("\n\n")

	visible := m.visibleItems()
	endIdx := min(m.scrollOffset+visible, len(m.categories))

	for i := m.scrollOffset; i < endIdx; i++ {
		cat := m.categories[i]
		b.WriteString(m.renderCategoryItemDesktop(cat, i == m.cursor, contentWidth-4))
		b.WriteString("\n")
	}

	if m.scrollOffset > 0 {
		b.WriteString(styles.TextMutedStyle.Render("     ↑ scroll up\n"))
	}
	if endIdx < len(m.categories) {
		b.WriteString(styles.TextMutedStyle.Render("     ↓ scroll down\n"))
	}

	// Summary box
	b.WriteString("\n")
	totalSize, count := m.calculateSelected()
	var summaryText string
	if count > 0 {
		summaryText = fmt.Sprintf("%s  •  %s to be freed",
			styles.SuccessStyle.Render(fmt.Sprintf("%d items selected", count)),
			styles.GetSizeStyle(totalSize).Render(components.FormatSize(totalSize)))
	} else {
		summaryText = styles.TextMutedStyle.Render("No items selected")
	}

	summaryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.SurfaceAlt).
		Padding(0, 2).
		Width(contentWidth).
		Render(summaryText)
	b.WriteString(summaryBox)

	return b.String()
}

func (m Model) renderCategoryItemMobile(cat CategoryData, selected bool) string {
	cursor := " "
	if selected {
		cursor = ">"
	}

	checkbox := "○"
	if cat.Selected {
		checkbox = "●"
	}

	name := cat.Info.Name
	if len(name) > 15 {
		name = name[:12] + "..."
	}
	paddedName := fmt.Sprintf("%-15s", name)

	size := components.FormatSizeAligned(cat.ScanResult.TotalSize)

	line := fmt.Sprintf("%s %s %s %s", cursor, checkbox, paddedName, size)

	if selected {
		return styles.SelectedItemStyle.Render(line)
	}
	if cat.Selected {
		return styles.CheckedItemStyle.Render(line)
	}
	return styles.ListItemStyle.Render(line)
}

func (m Model) renderCategoryItemDesktop(cat CategoryData, selected bool, width int) string {
	cursor := "  "
	if selected {
		cursor = "> "
	}

	checkbox := "[ ]"
	if cat.Selected {
		checkbox = "[✓]"
	}

	icon := cat.Info.Icon
	name := cat.Info.Name
	size := components.FormatSizeAligned(cat.ScanResult.TotalSize)
	sizeStyled := styles.GetSizeStyle(cat.ScanResult.TotalSize).Render(size)

	// Calculate padding for alignment (fixed size width = 9)
	nameWidth := width - len(cursor) - len(checkbox) - 4 - 9 - 6 // 4 for icon+space, 9 for size, 6 for spacing
	if nameWidth < 10 {
		nameWidth = 10
	}
	if len(name) > nameWidth {
		name = name[:nameWidth-3] + "..."
	}
	paddedName := fmt.Sprintf("%-*s", nameWidth, name)

	line := fmt.Sprintf("%s%s %s %s %s", cursor, checkbox, icon, paddedName, sizeStyled)

	if selected {
		return styles.SelectedItemStyle.Render(line)
	}
	if cat.Selected {
		return styles.CheckedItemStyle.Render(line)
	}
	return styles.ListItemStyle.Render(line)
}

func (m Model) renderDetailView() string {
	if styles.IsDesktop(m.width) {
		return m.renderDetailDesktop()
	}
	return m.renderDetailMobile()
}

func (m Model) renderDetailMobile() string {
	var b strings.Builder
	cat := m.categories[m.activeCategory]

	b.WriteString(styles.TitleStyle.Render(fmt.Sprintf(" %s %s", cat.Info.Icon, cat.Info.Name)))
	b.WriteString("\n\n")

	items := cat.ScanResult.Items
	if len(items) == 0 {
		b.WriteString(styles.TextMutedStyle.Render(" No items found"))
		b.WriteString("\n\n")
	} else {
		visible := m.visibleItems()
		endIdx := min(m.detailScrollOffset+visible, len(items))

		for i := m.detailScrollOffset; i < endIdx; i++ {
			item := items[i]
			cursor := " "
			if i == m.detailCursor {
				cursor = ">"
			}
			checkbox := "○"
			if item.IsSelected {
				checkbox = "●"
			}

			name := item.Name
			if len(name) > 20 {
				name = name[:17] + "..."
			}

			line := fmt.Sprintf("%s %s %s", cursor, checkbox, name)
			if i == m.detailCursor {
				b.WriteString(styles.SelectedItemStyle.Render(line))
			} else if item.IsSelected {
				b.WriteString(styles.CheckedItemStyle.Render(line))
			} else {
				b.WriteString(styles.ListItemStyle.Render(line))
			}
			b.WriteString("\n")
		}

		if endIdx < len(items) {
			b.WriteString(styles.TextMutedStyle.Render(" ↓ more\n"))
		}
	}

	return b.String()
}

func (m Model) renderDetailDesktop() string {
	var b strings.Builder
	cat := m.categories[m.activeCategory]
	contentWidth := styles.GetContentWidth(m.width)

	// Header
	header := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.SurfaceAlt).
		Padding(0, 2).
		Width(contentWidth).
		Render(styles.TitleStyle.Render(fmt.Sprintf("%s %s", cat.Info.Icon, cat.Info.Name)) +
			"  " + styles.SubtitleStyle.Render(cat.Info.Description))

	b.WriteString(header)
	b.WriteString("\n\n")

	items := cat.ScanResult.Items
	if len(items) == 0 {
		b.WriteString(styles.TextMutedStyle.Render("  No items found in this category\n"))
	} else {
		visible := m.visibleItems()
		if len(items) > visible {
			b.WriteString(styles.TextMutedStyle.Render(fmt.Sprintf("  Showing %d-%d of %d\n\n",
				m.detailScrollOffset+1,
				min(m.detailScrollOffset+visible, len(items)),
				len(items))))
		}

		endIdx := min(m.detailScrollOffset+visible, len(items))
		for i := m.detailScrollOffset; i < endIdx; i++ {
			item := items[i]
			b.WriteString(m.renderDetailItemDesktop(item, i == m.detailCursor, contentWidth-4))
			b.WriteString("\n")
		}

		if m.detailScrollOffset > 0 {
			b.WriteString(styles.TextMutedStyle.Render("     ↑ more above\n"))
		}
		if endIdx < len(items) {
			b.WriteString(styles.TextMutedStyle.Render("     ↓ more below\n"))
		}
	}

	return b.String()
}

func (m Model) renderDetailItemDesktop(item cleaner.CleanableItem, selected bool, width int) string {
	cursor := "  "
	if selected {
		cursor = "> "
	}

	checkbox := "[ ]"
	if item.IsSelected {
		checkbox = "[✓]"
	}

	danger := ""
	if item.IsDangerous {
		danger = styles.WarningStyle.Render(" !")
	}

	name := item.Name
	size := components.FormatSizeAligned(item.Size)
	sizeStyled := styles.GetSizeStyle(item.Size).Render(size)

	// Calculate name width (fixed size width = 9)
	nameWidth := width - len(cursor) - len(checkbox) - 9 - 6
	if nameWidth < 10 {
		nameWidth = 10
	}
	if len(name) > nameWidth {
		name = name[:nameWidth-3] + "..."
	}
	paddedName := fmt.Sprintf("%-*s", nameWidth, name)

	line := fmt.Sprintf("%s%s %s%s %s", cursor, checkbox, paddedName, danger, sizeStyled)

	if selected {
		return styles.SelectedItemStyle.Render(line)
	}
	if item.IsSelected {
		return styles.CheckedItemStyle.Render(line)
	}
	return styles.ListItemStyle.Render(line)
}

func (m Model) renderConfirmView() string {
	var b strings.Builder

	totalSize, count := m.calculateSelected()

	if styles.IsDesktop(m.width) {
		contentWidth := styles.GetContentWidth(m.width)

		// Warning dialog
		dialogContent := fmt.Sprintf("%s\n\nYou are about to clean %d categories (%s).\nThis action cannot be undone.",
			styles.WarningStyle.Render("⚠ Confirm Cleaning"),
			count,
			components.FormatSize(totalSize))

		dialog := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(styles.Orange).
			Padding(1, 3).
			Width(contentWidth).
			Render(dialogContent)

		b.WriteString(dialog)
		b.WriteString("\n\n")

		// Selected list
		b.WriteString(styles.HeaderStyle.Render("  Selected:"))
		b.WriteString("\n")
		for _, cat := range m.categories {
			if cat.Selected {
				b.WriteString(fmt.Sprintf("   • %s %s (%s)\n",
					cat.Info.Icon,
					cat.Info.Name,
					components.FormatSize(cat.ScanResult.TotalSize)))
			}
		}
	} else {
		// Mobile
		b.WriteString(styles.WarningStyle.Render(" ⚠ Confirm Clean"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf(" %d items • %s\n\n", count, components.FormatSize(totalSize)))
	}

	b.WriteString("\n")

	// Buttons
	cancelBtn := styles.ButtonStyle.Render(" Cancel ")
	confirmBtn := styles.DangerButtonStyle.Render(" Confirm ")
	if m.confirmCursor == 0 {
		cancelBtn = styles.ActiveButtonStyle.Render(" Cancel ")
	}

	b.WriteString("  " + cancelBtn + "  " + confirmBtn)

	return b.String()
}

func (m Model) renderPasswordView() string {
	var b strings.Builder

	if styles.IsDesktop(m.width) {
		contentWidth := styles.GetContentWidth(m.width)

		// Password dialog
		var dialogContent string
		if m.isValidating {
			dialogContent = fmt.Sprintf("%s\n\n%s Validating password...",
				styles.HeaderStyle.Render("🔐 Authentication Required"),
				m.spinner.View())
		} else {
			dialogContent = fmt.Sprintf("%s\n\nSome cleaning operations require administrator privileges.\nPlease enter your sudo password:",
				styles.HeaderStyle.Render("🔐 Authentication Required"))
		}

		dialog := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Magenta).
			Padding(1, 3).
			Width(contentWidth).
			Render(dialogContent)

		b.WriteString(dialog)
		b.WriteString("\n\n")

		// Password input field
		passwordDisplay := strings.Repeat("•", len(m.password))
		if len(passwordDisplay) == 0 {
			passwordDisplay = styles.TextMutedStyle.Render("(enter password)")
		}

		inputBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Cyan).
			Padding(0, 2).
			Width(contentWidth).
			Render("  Password: " + passwordDisplay)

		b.WriteString(inputBox)

		// Error message
		if m.passwordError != "" {
			b.WriteString("\n\n")
			b.WriteString("  " + styles.ErrorStyle.Render("✗ "+m.passwordError))
		}

	} else {
		// Mobile view
		b.WriteString(styles.HeaderStyle.Render(" 🔐 Authentication"))
		b.WriteString("\n\n")

		if m.isValidating {
			b.WriteString(fmt.Sprintf(" %s Validating...\n", m.spinner.View()))
		} else {
			b.WriteString(" Enter sudo password:\n\n")

			passwordDisplay := strings.Repeat("•", len(m.password))
			if len(passwordDisplay) == 0 {
				passwordDisplay = styles.TextMutedStyle.Render("(type here)")
			}
			b.WriteString(" > " + passwordDisplay + "\n")

			if m.passwordError != "" {
				b.WriteString("\n")
				b.WriteString(styles.ErrorStyle.Render(" ✗ " + m.passwordError))
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

func (m Model) renderCleaningView() string {
	var b strings.Builder

	if styles.IsDesktop(m.width) {
		contentWidth := styles.GetContentWidth(m.width)
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Cyan).
			Padding(2, 4).
			Width(contentWidth).
			Align(lipgloss.Center).
			Render(fmt.Sprintf("%s Cleaning in progress...\n\n%s",
				m.spinner.View(),
				styles.WarningStyle.Render("Please wait, do not interrupt.")))
		b.WriteString(box)
	} else {
		b.WriteString(fmt.Sprintf("\n %s Cleaning...\n", m.spinner.View()))
		b.WriteString(styles.WarningStyle.Render(" Please wait..."))
	}

	return b.String()
}

func (m Model) renderResultsView() string {
	var b strings.Builder

	var totalFreed int64
	var totalItems int
	for _, result := range m.cleanResults {
		totalFreed += result.BytesFreed
		totalItems += result.ItemsCleaned
	}

	if styles.IsDesktop(m.width) {
		contentWidth := styles.GetContentWidth(m.width)

		// Success header
		header := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Green).
			Padding(0, 2).
			Width(contentWidth).
			Render(styles.SuccessStyle.Render("✓ Cleaning Complete!") +
				fmt.Sprintf("  %d items cleaned, %s freed", totalItems, components.FormatSize(totalFreed)))

		b.WriteString(header)
		b.WriteString("\n\n")

		// Results
		for _, result := range m.cleanResults {
			b.WriteString("  " + components.RenderResult(result))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(styles.SuccessStyle.Render(" ✓ Complete!"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf(" %d items\n", totalItems))
		b.WriteString(fmt.Sprintf(" %s freed\n", components.FormatSize(totalFreed)))
	}

	return b.String()
}

func (m Model) calculateSelected() (int64, int) {
	var totalSize int64
	var count int
	for _, cat := range m.categories {
		if cat.Selected {
			totalSize += cat.ScanResult.TotalSize
			count++
		}
	}
	return totalSize, count
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Run() error {
	p := tea.NewProgram(
		New(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
