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
	"github.com/charmbracelet/x/ansi"
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
	confirmFrom    viewState // state to return to when canceling the confirm dialog

	scrollOffset       int
	detailScrollOffset int

	spinner spinner.Model

	scanner *scanner.Scanner
	cleaner *cleaner.Cleaner

	isScanning    bool
	isCleaning    bool
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
		// Show the scanning indicator while the initial scan runs
		isScanning: true,
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
	for i := range m.categories {
		cat := &m.categories[i]
		if cat.Selected {
			// Whole category selected: include (and mark) every item
			for j := range cat.ScanResult.Items {
				cat.ScanResult.Items[j].IsSelected = true
				items = append(items, cat.ScanResult.Items[j])
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
		// Only keep the spinner ticking while an operation is running
		// so the tick loop dies when idle.
		if m.isScanning || m.isCleaning || m.isValidating {
			return m, cmd
		}
		return m, nil
	case scanCompleteMsg:
		m.isScanning = false
		m.scanResults = msg.results
		m.initCategories()
		return m, nil
	case cleanCompleteMsg:
		m.isCleaning = false
		m.cleanResults = msg.results
		// Clear the sudo password from memory once cleaning is done
		m.password = ""
		m.passwordError = ""
		m.cleaner.SetPassword("")
		m.state = viewResults
		return m, nil
	case passwordValidMsg:
		m.isValidating = false
		if msg.valid {
			// Password is correct, proceed with cleaning
			m.isCleaning = true
			m.state = viewCleaning
			return m, tea.Batch(m.cleanSelected, m.spinner.Tick)
		}
		// Password is incorrect
		if msg.err != nil {
			m.passwordError = msg.err.Error()
		} else {
			m.passwordError = "Incorrect password"
		}
		m.password = ""
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
	// Reserve space for chrome (header bar, separators, footer).
	// Desktop shows more chrome than mobile.
	chrome := 8
	if styles.IsDesktop(m.width) {
		chrome = 10
	}
	available := m.height - chrome
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
		if m.hasSelectedCategories() || m.hasSelectedItems() {
			m.confirmCursor = 0
			m.confirmFrom = viewMain
			m.state = viewConfirm
		}
	case "r":
		m.isScanning = true
		return m, tea.Batch(m.scanCategories, m.spinner.Tick)
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
			m.confirmFrom = viewDetails
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
		m.state = m.confirmFrom
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
	return m, tea.Batch(m.cleanSelected, m.spinner.Tick)
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
			return m, tea.Batch(m.validatePassword, m.spinner.Tick)
		}
	case "backspace":
		if len(m.password) > 0 {
			m.password = m.password[:len(m.password)-1]
		}
	default:
		// Accept any printable character, including non-ASCII (UTF-8)
		if len(msg.Runes) > 0 {
			m.password += string(msg.Runes)
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
	case "r", "enter", "esc":
		m.isScanning = true
		m.state = viewMain
		return m, tea.Batch(m.scanCategories, m.spinner.Tick)
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

func (m Model) hasSelectedItems() bool {
	for _, cat := range m.categories {
		for _, item := range cat.ScanResult.Items {
			if item.IsSelected {
				return true
			}
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
	b.WriteString(styles.BadgeStyle.Render("\uf521 GOCLEAN")) // nf-fa-crown
	b.WriteString("\n\n")

	if m.isScanning {
		fmt.Fprintf(&b, " %s Scanning...\n", m.spinner.View())
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

// truncateDisplay truncates s to the given display width, appending an
// ellipsis. Safe for UTF-8 and wide characters.
func truncateDisplay(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	if w <= 1 {
		return ""
	}
	return ansi.Truncate(s, w, "...")
}

// padRight pads s with spaces until it occupies display width w.
func padRight(s string, w int) string {
	pad := w - lipgloss.Width(s)
	if pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// renderHeaderBar renders a badge-style title on the left and a
// right-aligned count, DiceSong style.
func renderHeaderBar(title, right string, width int) string {
	badge := styles.BadgeStyle.Render("\uf521 " + title) // nf-fa-crown
	rightText := styles.CountStyle.Render(right)
	gap := width - lipgloss.Width(badge) - lipgloss.Width(rightText)
	if gap < 1 {
		gap = 1
	}
	return badge + strings.Repeat(" ", gap) + rightText
}

// renderTableHeader renders gold column headers: "Name" left, "Size" right.
func renderTableHeader(width int) string {
	const sizeColWidth = 9
	nameColWidth := max(width-2-sizeColWidth, 10)
	return "  " +
		styles.TableHeaderStyle.Render(fmt.Sprintf("%-*s", nameColWidth, "Name")) +
		styles.TableHeaderStyle.Render(fmt.Sprintf("%*s", sizeColWidth, "Size"))
}

// renderFooter renders a separator and a help bar.
func renderFooter(width int, keys []components.HelpKey) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(styles.RenderSeparator(width))
	b.WriteString("\n")
	b.WriteString("  " + components.RenderHelp(keys))
	return b.String()
}

func (m Model) renderMainDesktop() string {
	var b strings.Builder
	contentWidth := styles.GetContentWidth(m.width)

	// Header bar: crown badge + right-aligned category count / position
	right := fmt.Sprintf("%d Categories", len(m.categories))
	if len(m.categories) > m.visibleItems() {
		right = fmt.Sprintf("%d / %d", m.cursor+1, len(m.categories))
	}
	b.WriteString(renderHeaderBar("GOCLEAN", right, contentWidth))
	b.WriteString("\n")
	b.WriteString(styles.RenderSeparator(contentWidth))
	b.WriteString("\n")

	if m.isScanning {
		b.WriteString(styles.PathBadgeStyle.Render(fmt.Sprintf("%s Scanning system...", m.spinner.View())))
		b.WriteString("\n")
		b.WriteString(styles.RenderSeparator(contentWidth))
		return b.String()
	}

	// Status line: selection summary badge
	totalSize, count := m.calculateSelected()
	var statusLine string
	if count > 0 {
		statusLine = styles.PathBadgeStyle.Render(fmt.Sprintf("%d selected", count)) +
			" " +
			styles.GetSizeStyle(totalSize).Render(components.FormatSize(totalSize)) +
			styles.TextMutedStyle.Render(" to be freed")
	} else {
		statusLine = styles.PathBadgeStyle.Render("Nothing selected") +
			" " +
			styles.TextMutedStyle.Render("space to select • a select all • n none")
	}
	b.WriteString(statusLine)
	b.WriteString("\n")
	b.WriteString(styles.RenderSeparator(contentWidth))
	b.WriteString("\n")

	// Table header
	b.WriteString(renderTableHeader(contentWidth))
	b.WriteString("\n")
	b.WriteString(styles.RenderSeparator(contentWidth))
	b.WriteString("\n")

	// Categories
	visible := m.visibleItems()
	endIdx := min(m.scrollOffset+visible, len(m.categories))
	for i := m.scrollOffset; i < endIdx; i++ {
		b.WriteString(m.renderCategoryItemDesktop(m.categories[i], i == m.cursor, contentWidth))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(renderFooter(contentWidth, []components.HelpKey{
		{Key: "↑↓", Desc: "move"}, {Key: "space", Desc: "select"}, {Key: "enter", Desc: "details"},
		{Key: "c", Desc: "clean"}, {Key: "r", Desc: "rescan"}, {Key: "q", Desc: "quit"},
	}))

	return b.String()
}

func (m Model) renderCategoryItemMobile(cat CategoryData, selected bool) string {
	cursor := " "
	if selected {
		cursor = "▶"
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
		cursor = "▶ "
	}

	checkbox := "[ ]"
	if cat.Selected {
		checkbox = "[✓]"
	}

	const sizeColWidth = 9
	prefix := cursor + checkbox + " " + cat.Info.Icon + " "
	prefixWidth := lipgloss.Width(prefix)

	name := cat.Info.Name
	nameWidth := max(width-prefixWidth-sizeColWidth, 10)
	paddedName := padRight(truncateDisplay(name, nameWidth), nameWidth)

	size := components.FormatSizeAligned(cat.ScanResult.TotalSize)
	// Selected rows are colored entirely gold by SelectedItemStyle
	sizeStyled := size
	if !selected {
		sizeStyled = styles.GetSizeStyle(cat.ScanResult.TotalSize).Render(size)
	}

	line := prefix + paddedName + sizeStyled

	if selected {
		return styles.SelectedItemStyle.Width(width).Render(line)
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

	b.WriteString(styles.PathBadgeStyle.Render(fmt.Sprintf("%s %s", cat.Info.Icon, cat.Info.Name)))
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
				cursor = "▶"
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
	items := cat.ScanResult.Items

	// Header bar with position indicator
	b.WriteString(renderHeaderBar("GOCLEAN", fmt.Sprintf("%d / %d", m.activeCategory+1, len(m.categories)), contentWidth))
	b.WriteString("\n")
	b.WriteString(styles.RenderSeparator(contentWidth))
	b.WriteString("\n")

	// Category badge + muted description
	b.WriteString(styles.PathBadgeStyle.Render(cat.Info.Icon + " " + cat.Info.Name))
	b.WriteString(styles.TextMutedStyle.Render("  " + cat.Info.Description))
	b.WriteString("\n")
	b.WriteString(styles.RenderSeparator(contentWidth))
	b.WriteString("\n")

	// Table header
	b.WriteString(renderTableHeader(contentWidth))
	b.WriteString("\n")
	b.WriteString(styles.RenderSeparator(contentWidth))
	b.WriteString("\n")

	if len(items) == 0 {
		b.WriteString(styles.TextMutedStyle.Render("  No items found in this category"))
	} else {
		visible := m.visibleItems()
		endIdx := min(m.detailScrollOffset+visible, len(items))
		for i := m.detailScrollOffset; i < endIdx; i++ {
			b.WriteString(m.renderDetailItemDesktop(items[i], i == m.detailCursor, contentWidth))
			b.WriteString("\n")
		}

		// Compact scroll indicator
		if m.detailScrollOffset > 0 || endIdx < len(items) {
			var hints []string
			if m.detailScrollOffset > 0 {
				hints = append(hints, "↑ more above")
			}
			if endIdx < len(items) {
				hints = append(hints, "↓ more below")
			}
			b.WriteString(styles.TextMutedStyle.Render("  " + strings.Join(hints, "  ")))
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString(renderFooter(contentWidth, []components.HelpKey{
		{Key: "space", Desc: "select"}, {Key: "a", Desc: "all"}, {Key: "n", Desc: "none"},
		{Key: "c", Desc: "clean"}, {Key: "esc", Desc: "back"},
	}))

	return b.String()
}

func (m Model) renderDetailItemDesktop(item cleaner.CleanableItem, selected bool, width int) string {
	cursor := "  "
	if selected {
		cursor = "▶ "
	}

	checkbox := "[ ]"
	if item.IsSelected {
		checkbox = "[✓]"
	}

	danger := ""
	dangerWidth := 0
	if item.IsDangerous {
		danger = styles.WarningStyle.Render(" !")
		dangerWidth = 2
	}

	const sizeColWidth = 9
	prefix := cursor + checkbox + " "
	prefixWidth := lipgloss.Width(prefix)

	name := item.Name
	nameWidth := max(width-prefixWidth-dangerWidth-1-sizeColWidth, 10)
	paddedName := padRight(truncateDisplay(name, nameWidth), nameWidth)

	size := components.FormatSizeAligned(item.Size)
	// Selected rows are colored entirely gold by SelectedItemStyle
	sizeStyled := size
	if !selected {
		sizeStyled = styles.GetSizeStyle(item.Size).Render(size)
	}

	line := prefix + paddedName + danger + " " + sizeStyled

	if selected {
		return styles.SelectedItemStyle.Width(width).Render(line)
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

		// Collect selection summary: whole categories and/or individual items
		type selectedCat struct {
			icon, name string
			size       int64
		}
		var selected []selectedCat
		for _, cat := range m.categories {
			var size int64
			var n int
			if cat.Selected {
				size = cat.ScanResult.TotalSize
				n = len(cat.ScanResult.Items)
			} else {
				for _, item := range cat.ScanResult.Items {
					if item.IsSelected {
						size += item.Size
						n++
					}
				}
			}
			if n > 0 {
				selected = append(selected, selectedCat{icon: cat.Info.Icon, name: cat.Info.Name, size: size})
			}
		}

		// Warning dialog
		dialogContent := fmt.Sprintf("%s\n\nYou are about to clean %d categories (%s).\nThis action cannot be undone.",
			styles.WarningStyle.Render("\uf071 Confirm Cleaning"), // nf-fa-warning
			len(selected),
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
		for _, s := range selected {
			fmt.Fprintf(&b, "   • %s %s (%s)\n",
				s.icon,
				s.name,
				components.FormatSize(s.size))
		}
	} else {
		// Mobile
		b.WriteString(styles.WarningStyle.Render(" \uf071 Confirm Clean")) // nf-fa-warning
		b.WriteString("\n\n")
		fmt.Fprintf(&b, " %d items • %s\n\n", count, components.FormatSize(totalSize))
	}

	b.WriteString("\n")

	// Buttons
	cancelBtn := styles.ButtonStyle.Render(" Cancel ")
	confirmBtn := styles.DangerButtonStyle.Render(" Confirm ")
	if m.confirmCursor == 0 {
		cancelBtn = styles.ActiveButtonStyle.Render(" Cancel ")
	}

	b.WriteString("  ")
	b.WriteString(cancelBtn)
	b.WriteString("  ")
	b.WriteString(confirmBtn)
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
				styles.HeaderStyle.Render("\uf023 Authentication Required"), // nf-fa-lock
				m.spinner.View())
		} else {
			dialogContent = fmt.Sprintf("%s\n\nSome cleaning operations require administrator privileges.\nPlease enter your sudo password:",
				styles.HeaderStyle.Render("\uf023 Authentication Required")) // nf-fa-lock
		}

		dialog := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Blue).
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
			BorderForeground(styles.Blue).
			Padding(0, 2).
			Width(contentWidth).
			Render("  Password: " + passwordDisplay)

		b.WriteString(inputBox)

		// Error message
		if m.passwordError != "" {
			b.WriteString("\n\n")
			b.WriteString("  ")
			b.WriteString(styles.ErrorStyle.Render("\uf00d " + m.passwordError)) // nf-fa-times
		}

	} else {
		// Mobile view
		b.WriteString(styles.HeaderStyle.Render(" \uf023 Authentication")) // nf-fa-lock
		b.WriteString("\n\n")

		if m.isValidating {
			fmt.Fprintf(&b, " %s Validating...\n", m.spinner.View())
		} else {
			b.WriteString(" Enter sudo password:\n\n")

			passwordDisplay := strings.Repeat("•", len(m.password))
			if len(passwordDisplay) == 0 {
				passwordDisplay = styles.TextMutedStyle.Render("(type here)")
			}
			b.WriteString(" > ")
			b.WriteString(passwordDisplay)
			b.WriteString("\n")

			if m.passwordError != "" {
				b.WriteString("\n")
				b.WriteString(styles.ErrorStyle.Render(" \uf00d " + m.passwordError)) // nf-fa-times
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
			BorderForeground(styles.Yellow).
			Padding(2, 4).
			Width(contentWidth).
			Align(lipgloss.Center).
			Render(fmt.Sprintf("%s Cleaning in progress...\n\n%s",
				m.spinner.View(),
				styles.WarningStyle.Render("Please wait, do not interrupt.")))
		b.WriteString(box)
	} else {
		fmt.Fprintf(&b, "\n %s Cleaning...\n", m.spinner.View())
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
			Render(styles.SuccessStyle.Render("\uf00c Cleaning Complete!") + // nf-fa-check
				fmt.Sprintf("  %d items cleaned, %s freed", totalItems, components.FormatSize(totalFreed)))

		b.WriteString(header)
		b.WriteString("\n\n")

		// Results
		for _, result := range m.cleanResults {
			b.WriteString("  ")
			b.WriteString(components.RenderResult(result))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(styles.SuccessStyle.Render(" \uf00c Complete!")) // nf-fa-check
		b.WriteString("\n\n")
		fmt.Fprintf(&b, " %d items\n", totalItems)
		fmt.Fprintf(&b, " %s freed\n", components.FormatSize(totalFreed))
	}

	return b.String()
}

func (m Model) calculateSelected() (int64, int) {
	var totalSize int64
	var count int
	for _, cat := range m.categories {
		if cat.Selected {
			// Whole category selected: all its items are included
			totalSize += cat.ScanResult.TotalSize
			count++
			continue
		}
		// Individually selected items still count
		for _, item := range cat.ScanResult.Items {
			if item.IsSelected {
				totalSize += item.Size
				count++
			}
		}
	}
	return totalSize, count
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
