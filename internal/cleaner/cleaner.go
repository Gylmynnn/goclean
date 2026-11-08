package cleaner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Cleaner handles all cleaning operations
type Cleaner struct {
	options  CleanOptions
	password string
}

// New creates a new Cleaner instance
func New(opts CleanOptions) *Cleaner {
	return &Cleaner{options: opts}
}

// SetPassword sets the sudo password
func (c *Cleaner) SetPassword(password string) {
	c.password = password
}

// NeedsRoot checks if any category requires root
func NeedsRoot(categories []CleanCategory) bool {
	rootCategories := map[CleanCategory]bool{
		CategoryPackageCache:   true,
		CategoryOrphanPackages: true,
		CategorySystemCache:    true,
		CategoryLogs:           true,
		CategoryTempFiles:      true,
	}
	for _, cat := range categories {
		if rootCategories[cat] {
			return true
		}
	}
	return false
}

// ValidatePassword checks if the sudo password is correct
func (c *Cleaner) ValidatePassword() error {
	if c.password == "" {
		return fmt.Errorf("password is empty")
	}

	if _, err := exec.LookPath("sudo"); err != nil {
		return fmt.Errorf("sudo not found in PATH")
	}

	cmd := exec.Command("sudo", "-S", "-v")
	cmd.Stdin = strings.NewReader(c.password + "\n")
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// sudo -v exits 1 on failed authentication
			return fmt.Errorf("incorrect password")
		}
		return fmt.Errorf("sudo failed: %v", err)
	}
	return nil
}

// Clean performs the cleaning operation for selected items
func (c *Cleaner) Clean(items []CleanableItem) []CleanResult {
	categoryItems := make(map[CleanCategory][]CleanableItem)
	for _, item := range items {
		if item.IsSelected {
			categoryItems[item.Category] = append(categoryItems[item.Category], item)
		}
	}

	var results []CleanResult
	// Iterate in the fixed category order so results are deterministic
	for _, cat := range GetCategoryInfo() {
		if itemList, ok := categoryItems[cat.Category]; ok {
			results = append(results, c.cleanCategory(cat.Category, itemList))
		}
	}

	return results
}

func (c *Cleaner) runSudoCommand(name string, args ...string) ([]byte, error) {
	// Force a C locale so paccache/journalctl output stays parseable
	// regardless of the user's system language.
	fullArgs := append([]string{"-S", "LC_ALL=C", "LC_MESSAGES=C", name}, args...)
	cmd := exec.Command("sudo", fullArgs...)
	cmd.Stdin = strings.NewReader(c.password + "\n")
	return cmd.CombinedOutput()
}

func (c *Cleaner) cleanCategory(category CleanCategory, items []CleanableItem) CleanResult {
	result := CleanResult{
		Category: category,
		Success:  true,
	}

	switch category {
	case CategoryPackageCache:
		return c.cleanPackageCache()
	case CategoryOrphanPackages:
		return c.cleanOrphanPackages(items)
	case CategorySystemCache:
		return c.cleanSystemCache(items)
	case CategoryUserCache:
		return c.cleanUserCache(items)
	case CategoryLogs:
		return c.cleanLogs()
	case CategoryThumbnails:
		return c.cleanThumbnails()
	case CategoryTrash:
		return c.cleanTrash()
	case CategoryTempFiles:
		return c.cleanTempFiles(items)
	default:
		result.Success = false
		result.Errors = append(result.Errors, fmt.Errorf("unknown category: %s", category))
	}

	return result
}

func (c *Cleaner) cleanPackageCache() CleanResult {
	result := CleanResult{Category: CategoryPackageCache, Success: true}

	if c.options.DryRun {
		return result
	}

	keepN := c.options.KeepLastN
	if keepN <= 0 {
		keepN = 1
	}

	output, err := c.runSudoCommand("paccache", "-r", "-k", fmt.Sprintf("%d", keepN))
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Errorf("paccache error: %v - %s", err, string(output)))
	} else {
		// Parse "==> finished: N packages found (no change to M), X GiB freed"
		result.ItemsCleaned += int(parsePaccacheCleaned(string(output)))
		result.BytesFreed += parseTrailingSize(string(output), "freed")
	}

	output, err = c.runSudoCommand("paccache", "-ruk0")
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Errorf("paccache uninstalled error: %v - %s", err, string(output)))
	} else {
		result.ItemsCleaned += int(parsePaccacheCleaned(string(output)))
		result.BytesFreed += parseTrailingSize(string(output), "freed")
	}

	return result
}

func (c *Cleaner) cleanOrphanPackages(items []CleanableItem) CleanResult {
	result := CleanResult{Category: CategoryOrphanPackages, Success: true}

	if c.options.DryRun {
		return result
	}

	// Only remove the packages the user actually selected
	selected := make(map[string]bool, len(items))
	for _, item := range items {
		if item.IsSelected {
			selected[item.Name] = true
		}
	}
	if len(selected) == 0 {
		return result
	}

	cmd := exec.Command("pacman", "-Qdtq")
	output, err := cmd.Output()
	if err != nil && len(output) == 0 {
		return result
	}

	orphans := strings.TrimSpace(string(output))
	if orphans == "" {
		return result
	}

	var packages []string
	for pkg := range strings.SplitSeq(orphans, "\n") {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}
		if selected[pkg] {
			packages = append(packages, pkg)
		}
	}
	if len(packages) == 0 {
		return result
	}

	args := append([]string{"-Rns", "--noconfirm"}, packages...)
	output, err = c.runSudoCommand("pacman", args...)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Errorf("pacman remove error: %v - %s", err, string(output)))
		return result
	}

	result.ItemsCleaned = len(packages)
	return result
}

func (c *Cleaner) cleanSystemCache(items []CleanableItem) CleanResult {
	return c.cleanPathsWithSudo(CategorySystemCache, items)
}

func (c *Cleaner) cleanUserCache(items []CleanableItem) CleanResult {
	return c.cleanPaths(CategoryUserCache, items)
}

func (c *Cleaner) cleanLogs() CleanResult {
	result := CleanResult{Category: CategoryLogs, Success: true}

	if c.options.DryRun {
		return result
	}

	output, err := c.runSudoCommand("journalctl", "--vacuum-time=3d")
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Errorf("journalctl error: %v - %s", err, string(output)))
	} else {
		// Parse "Vacuuming done, freed 1.2G of archived journals from ..."
		result.BytesFreed = parseLeadingSize(string(output), "freed")
	}

	return result
}

func (c *Cleaner) cleanThumbnails() CleanResult {
	result := CleanResult{Category: CategoryThumbnails, Success: true}

	if c.options.DryRun {
		return result
	}

	home, err := os.UserHomeDir()
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, err)
		return result
	}

	thumbPath := filepath.Join(home, ".cache", "thumbnails")
	size, count, err := c.removeDirContents(thumbPath)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, err)
	}

	result.BytesFreed = size
	result.ItemsCleaned = count
	return result
}

func (c *Cleaner) cleanTrash() CleanResult {
	result := CleanResult{Category: CategoryTrash, Success: true}

	if c.options.DryRun {
		return result
	}

	home, err := os.UserHomeDir()
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, err)
		return result
	}

	trashPaths := []string{
		filepath.Join(home, ".local", "share", "Trash", "files"),
		filepath.Join(home, ".local", "share", "Trash", "info"),
	}

	// Only contents of "files" count as cleaned items; the metadata in
	// "info" is deleted too but must not inflate the item count.
	size, count, err := c.removeDirContents(trashPaths[0])
	if err != nil {
		result.Errors = append(result.Errors, err)
	}

	if err := c.clearDirContents(trashPaths[1]); err != nil {
		result.Errors = append(result.Errors, err)
	}

	result.BytesFreed = size
	result.ItemsCleaned = count
	return result
}

func (c *Cleaner) cleanTempFiles(items []CleanableItem) CleanResult {
	return c.cleanPathsWithSudo(CategoryTempFiles, items)
}

func (c *Cleaner) cleanPathsWithSudo(category CleanCategory, items []CleanableItem) CleanResult {
	result := CleanResult{Category: category, Success: true}

	if c.options.DryRun {
		return result
	}

	for _, item := range items {
		if !item.IsSelected {
			continue
		}

		// Never touch protected system paths, even if listed
		if isProtectedPath(item.Path) {
			continue
		}

		if item.IsDangerous && !c.options.IncludeDangerous {
			continue
		}

		output, err := c.runSudoCommand("rm", "-rf", item.Path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to remove %s: %v - %s", item.Path, err, strings.TrimSpace(string(output))))
			continue
		}

		result.BytesFreed += item.Size
		result.ItemsCleaned++
	}

	if len(result.Errors) > 0 && result.ItemsCleaned == 0 {
		result.Success = false
	}

	return result
}

func (c *Cleaner) cleanPaths(category CleanCategory, items []CleanableItem) CleanResult {
	result := CleanResult{Category: category, Success: true}

	if c.options.DryRun {
		return result
	}

	for _, item := range items {
		if !item.IsSelected {
			continue
		}

		// Never touch protected system paths, even if listed
		if isProtectedPath(item.Path) {
			continue
		}

		if item.IsDangerous && !c.options.IncludeDangerous {
			continue
		}

		err := os.RemoveAll(item.Path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to remove %s: %v", item.Path, err))
			continue
		}

		result.BytesFreed += item.Size
		result.ItemsCleaned++
	}

	if len(result.Errors) > 0 && result.ItemsCleaned == 0 {
		result.Success = false
	}

	return result
}

func (c *Cleaner) removeDirContents(dir string) (int64, int, error) {
	var totalSize int64
	var count int

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if entry.IsDir() {
			size, subCount, _ := c.getDirSize(path)
			totalSize += size
			count += subCount
		} else {
			totalSize += info.Size()
			count++
		}

		if err := os.RemoveAll(path); err != nil {
			continue
		}
	}

	return totalSize, count, nil
}

// clearDirContents removes everything inside dir without counting entries
// (used for Trash metadata).
func (c *Cleaner) clearDirContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var firstErr error
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// protectedTempNames are special directories inside /tmp that must never
// be removed (X11/Wayland sockets and friends).
var protectedTempNames = map[string]bool{
	".X11-unix":  true,
	".ICE-unix":  true,
	".XIM-unix":  true,
	".font-unix": true,
	".Test-unix": true,
}

// isProtectedPath reports whether path must never be removed by the cleaner.
func isProtectedPath(path string) bool {
	clean := filepath.Clean(path)
	if clean == "/" || clean == "/tmp" || clean == "/var/tmp" {
		return true
	}
	base := filepath.Base(clean)
	if protectedTempNames[base] || base == "snap-private-tmp" {
		return true
	}
	return strings.HasPrefix(base, "systemd-private-")
}

// tryParseSize parses a size token like "2.5GiB", "1.2G" or "0B" into bytes.
func tryParseSize(token string) int64 {
	token = strings.ToUpper(strings.TrimSpace(token))
	i := 0
	for i < len(token) && (token[i] >= '0' && token[i] <= '9' || token[i] == '.') {
		i++
	}
	if i == 0 {
		return 0
	}
	value, err := strconv.ParseFloat(token[:i], 64)
	if err != nil {
		return 0
	}
	mult := float64(1)
	switch {
	case strings.HasPrefix(token[i:], "T"):
		mult = 1 << 40
	case strings.HasPrefix(token[i:], "G"):
		mult = 1 << 30
	case strings.HasPrefix(token[i:], "M"):
		mult = 1 << 20
	case strings.HasPrefix(token[i:], "K"):
		mult = 1 << 10
	}
	return int64(value * mult)
}

// parseTrailingSize finds a size written just before the keyword, as in
// paccache output: "==> finished: 45 packages found (no change to 0), 2.5 GiB freed".
func parseTrailingSize(output, keyword string) int64 {
	for _, line := range strings.Split(output, "\n") {
		idx := strings.Index(line, keyword)
		if idx < 0 {
			continue
		}
		fields := strings.Fields(line[:idx])
		if len(fields) == 0 {
			continue
		}
		// Unit may be a separate token: "2.5 GiB"
		if len(fields) >= 2 {
			if v := tryParseSize(fields[len(fields)-2] + fields[len(fields)-1]); v > 0 {
				return v
			}
		}
		if v := tryParseSize(fields[len(fields)-1]); v > 0 {
			return v
		}
	}
	return 0
}

// parseLeadingSize finds a size written just after the keyword, as in
// journalctl output: "Vacuuming done, freed 1.2G of archived journals from ...".
func parseLeadingSize(output, keyword string) int64 {
	for _, line := range strings.Split(output, "\n") {
		idx := strings.Index(line, keyword)
		if idx < 0 {
			continue
		}
		for _, field := range strings.Fields(line[idx+len(keyword):]) {
			if len(field) > 0 && field[0] >= '0' && field[0] <= '9' {
				return tryParseSize(field)
			}
		}
	}
	return 0
}

// parsePaccacheCleaned extracts "N packages found (no change to M)" and
// returns N-M, the number of packages paccache actually removed.
func parsePaccacheCleaned(output string) int64 {
	for _, line := range strings.Split(output, "\n") {
		idx := strings.Index(line, "packages found")
		if idx < 0 {
			continue
		}
		found := trailingNumber(line[:idx])
		noChange := int64(0)
		if start := strings.Index(line, "(no change to "); start >= 0 {
			noChange = leadingNumber(line[start+len("(no change to "):])
		}
		if cleaned := found - noChange; cleaned > 0 {
			return cleaned
		}
		return 0
	}
	return 0
}

func trailingNumber(s string) int64 {
	s = strings.TrimSpace(s)
	i := len(s)
	for i > 0 && s[i-1] >= '0' && s[i-1] <= '9' {
		i--
	}
	if i == len(s) {
		return 0
	}
	v, _ := strconv.Atoi(s[i:])
	return int64(v)
}

func leadingNumber(s string) int64 {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0
	}
	v, _ := strconv.Atoi(s[:i])
	return int64(v)
}

func (c *Cleaner) getDirSize(path string) (int64, int, error) {
	var size int64
	var count int

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
			count++
		}
		return nil
	})

	return size, count, err
}
