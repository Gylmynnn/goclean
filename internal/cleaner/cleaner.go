package cleaner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	cmd := exec.Command("sudo", "-S", "-v")
	cmd.Stdin = strings.NewReader(c.password + "\n")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("invalid password")
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
	for category, categoryItemList := range categoryItems {
		result := c.cleanCategory(category, categoryItemList)
		results = append(results, result)
	}

	return results
}

func (c *Cleaner) runSudoCommand(name string, args ...string) ([]byte, error) {
	fullArgs := append([]string{"-S", name}, args...)
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
		return c.cleanOrphanPackages()
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
	}

	output, err = c.runSudoCommand("paccache", "-ruk0")
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("paccache uninstalled error: %v - %s", err, string(output)))
	}

	return result
}

func (c *Cleaner) cleanOrphanPackages() CleanResult {
	result := CleanResult{Category: CategoryOrphanPackages, Success: true}

	if c.options.DryRun {
		return result
	}

	cmd := exec.Command("pacman", "-Qdtq")
	output, err := cmd.Output()
	if err != nil {
		if len(output) == 0 {
			return result
		}
	}

	orphans := strings.TrimSpace(string(output))
	if orphans == "" {
		return result
	}

	packages := strings.Split(orphans, "\n")
	args := append([]string{"-Rns", "--noconfirm"}, packages...)
	output, err = c.runSudoCommand("pacman", args...)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Errorf("pacman remove error: %v - %s", err, string(output)))
	}

	result.ItemsCleaned = len(packages)
	return result
}

func (c *Cleaner) cleanSystemCache(items []CleanableItem) CleanResult {
	return c.cleanPaths(CategorySystemCache, items)
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

	var totalSize int64
	var totalCount int

	for _, trashPath := range trashPaths {
		size, count, err := c.removeDirContents(trashPath)
		if err != nil {
			result.Errors = append(result.Errors, err)
		}
		totalSize += size
		totalCount += count
	}

	result.BytesFreed = totalSize
	result.ItemsCleaned = totalCount
	return result
}

func (c *Cleaner) cleanTempFiles(items []CleanableItem) CleanResult {
	return c.cleanPaths(CategoryTempFiles, items)
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
