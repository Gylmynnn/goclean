package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Gylmynnn/goclean/internal/cleaner"
)

// Scanner handles scanning for cleanable items
type Scanner struct{}

// New creates a new Scanner instance
func New() *Scanner {
	return &Scanner{}
}

// ScanAll scans all categories and returns results
func (s *Scanner) ScanAll() []cleaner.ScanResult {
	categories := cleaner.GetCategoryInfo()
	results := make([]cleaner.ScanResult, 0, len(categories))

	for _, cat := range categories {
		result := s.ScanCategory(cat.Category)
		results = append(results, result)
	}

	return results
}

// ScanCategory scans a specific category
func (s *Scanner) ScanCategory(category cleaner.CleanCategory) cleaner.ScanResult {
	switch category {
	case cleaner.CategoryPackageCache:
		return s.scanPackageCache()
	case cleaner.CategoryOrphanPackages:
		return s.scanOrphanPackages()
	case cleaner.CategorySystemCache:
		return s.scanSystemCache()
	case cleaner.CategoryUserCache:
		return s.scanUserCache()
	case cleaner.CategoryLogs:
		return s.scanLogs()
	case cleaner.CategoryThumbnails:
		return s.scanThumbnails()
	case cleaner.CategoryTrash:
		return s.scanTrash()
	case cleaner.CategoryTempFiles:
		return s.scanTempFiles()
	default:
		return cleaner.ScanResult{Category: category}
	}
}

func (s *Scanner) scanPackageCache() cleaner.ScanResult {
	result := cleaner.ScanResult{Category: cleaner.CategoryPackageCache}

	cachePath := "/var/cache/pacman/pkg"
	items, totalSize := s.scanDirectory(cachePath, cleaner.CategoryPackageCache, false)

	result.Items = items
	result.TotalSize = totalSize
	result.TotalCount = len(items)

	return result
}

func (s *Scanner) scanOrphanPackages() cleaner.ScanResult {
	result := cleaner.ScanResult{Category: cleaner.CategoryOrphanPackages}

	cmd := exec.Command("pacman", "-Qdtq")
	output, err := cmd.Output()
	if err != nil {
		// No orphans is not an error
		return result
	}

	orphans := strings.TrimSpace(string(output))
	if orphans == "" {
		return result
	}

	packages := strings.SplitSeq(orphans, "\n")
	for pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}

		// Get package size
		sizeCmd := exec.Command("pacman", "-Qi", pkg)
		sizeOutput, _ := sizeCmd.Output()
		size := s.parsePackageSize(string(sizeOutput))

		item := cleaner.CleanableItem{
			Name:        pkg,
			Path:        pkg, 
			Size:        size,
			Category:    cleaner.CategoryOrphanPackages,
			Description: "Orphan package - no longer required as dependency",
			IsSelected:  false,
			IsDangerous: true,
		}
		result.Items = append(result.Items, item)
		result.TotalSize += size
	}

	result.TotalCount = len(result.Items)
	return result
}

func (s *Scanner) parsePackageSize(output string) int64 {
	lines := strings.SplitSeq(output, "\n")
	for line := range lines {
		if strings.HasPrefix(line, "Installed Size") {
			parts := strings.Split(line, ":")
			if len(parts) < 2 {
				continue
			}
			sizeStr := strings.TrimSpace(parts[1])
			return s.parseSizeString(sizeStr)
		}
	}
	return 0
}

func (s *Scanner) parseSizeString(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(sizeStr)
	parts := strings.Fields(sizeStr)
	if len(parts) < 2 {
		return 0
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToUpper(parts[1])
	switch {
	case strings.HasPrefix(unit, "K"):
		return int64(value * 1024)
	case strings.HasPrefix(unit, "M"):
		return int64(value * 1024 * 1024)
	case strings.HasPrefix(unit, "G"):
		return int64(value * 1024 * 1024 * 1024)
	default:
		return int64(value)
	}
}

func (s *Scanner) scanSystemCache() cleaner.ScanResult {
	result := cleaner.ScanResult{Category: cleaner.CategorySystemCache}

	cachePaths := []string{
		"/var/cache",
	}

	for _, path := range cachePaths {
		items, size := s.scanDirectory(path, cleaner.CategorySystemCache, false)
		result.Items = append(result.Items, items...)
		result.TotalSize += size
	}

	result.TotalCount = len(result.Items)
	return result
}

func (s *Scanner) scanUserCache() cleaner.ScanResult {
	result := cleaner.ScanResult{Category: cleaner.CategoryUserCache}

	home, err := os.UserHomeDir()
	if err != nil {
		result.Error = err
		return result
	}

	cachePath := filepath.Join(home, ".cache")
	items, totalSize := s.scanDirectory(cachePath, cleaner.CategoryUserCache, false)

	result.Items = items
	result.TotalSize = totalSize
	result.TotalCount = len(items)

	return result
}

func (s *Scanner) scanLogs() cleaner.ScanResult {
	result := cleaner.ScanResult{Category: cleaner.CategoryLogs}

	// Get journalctl disk usage
	cmd := exec.Command("journalctl", "--disk-usage")
	output, err := cmd.Output()
	if err == nil {
		// Parse output like "Archived and active journals take up 256.0M in the file system."
		outputStr := string(output)
		size := s.parseJournalSize(outputStr)

		item := cleaner.CleanableItem{
			Name:        "Journal Logs",
			Path:        "/var/log/journal",
			Size:        size,
			Category:    cleaner.CategoryLogs,
			Description: "Systemd journal logs",
			IsSelected:  false,
			IsDangerous: false,
		}
		result.Items = append(result.Items, item)
		result.TotalSize += size
	}

	// Scan /var/log for old log files
	logItems, logSize := s.scanLogFiles("/var/log")
	result.Items = append(result.Items, logItems...)
	result.TotalSize += logSize

	result.TotalCount = len(result.Items)
	return result
}

func (s *Scanner) parseJournalSize(output string) int64 {
	// Look for size pattern like "256.0M" or "1.2G"
	words := strings.FieldsSeq(output)
	for word := range words {
		if len(word) > 1 {
			lastChar := word[len(word)-1]
			if lastChar == 'M' || lastChar == 'G' || lastChar == 'K' || lastChar == 'B' {
				return s.parseSizeString(word[:len(word)-1] + " " + string(lastChar) + "iB")
			}
		}
	}
	return 0
}

func (s *Scanner) scanLogFiles(logPath string) ([]cleaner.CleanableItem, int64) {
	var items []cleaner.CleanableItem
	var totalSize int64

	entries, err := os.ReadDir(logPath)
	if err != nil {
		return items, totalSize
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Look for rotated log files (*.gz, *.old, *.1, etc.)
		if strings.HasSuffix(name, ".gz") ||
			strings.HasSuffix(name, ".old") ||
			strings.HasSuffix(name, ".1") ||
			strings.HasSuffix(name, ".2") {

			info, err := entry.Info()
			if err != nil {
				continue
			}

			path := filepath.Join(logPath, name)
			item := cleaner.CleanableItem{
				Name:        name,
				Path:        path,
				Size:        info.Size(),
				Category:    cleaner.CategoryLogs,
				Description: "Rotated log file",
				LastAccess:  info.ModTime(),
				IsSelected:  false,
				IsDangerous: false,
			}
			items = append(items, item)
			totalSize += info.Size()
		}
	}

	return items, totalSize
}

func (s *Scanner) scanThumbnails() cleaner.ScanResult {
	result := cleaner.ScanResult{Category: cleaner.CategoryThumbnails}

	home, err := os.UserHomeDir()
	if err != nil {
		result.Error = err
		return result
	}

	thumbPath := filepath.Join(home, ".cache", "thumbnails")
	size := s.getDirSize(thumbPath)

	if size > 0 {
		item := cleaner.CleanableItem{
			Name:        "Thumbnails",
			Path:        thumbPath,
			Size:        size,
			Category:    cleaner.CategoryThumbnails,
			Description: "Cached thumbnail images",
			IsSelected:  false,
			IsDangerous: false,
		}
		result.Items = append(result.Items, item)
		result.TotalSize = size
		result.TotalCount = 1
	}

	return result
}

func (s *Scanner) scanTrash() cleaner.ScanResult {
	result := cleaner.ScanResult{Category: cleaner.CategoryTrash}

	home, err := os.UserHomeDir()
	if err != nil {
		result.Error = err
		return result
	}

	trashPath := filepath.Join(home, ".local", "share", "Trash", "files")
	size := s.getDirSize(trashPath)

	// Count items in trash
	entries, _ := os.ReadDir(trashPath)
	count := len(entries)

	if size > 0 || count > 0 {
		item := cleaner.CleanableItem{
			Name:        "Trash",
			Path:        trashPath,
			Size:        size,
			Category:    cleaner.CategoryTrash,
			Description: strings.ReplaceAll("Contains %d items", "%d", strconv.Itoa(count)),
			IsSelected:  false,
			IsDangerous: false,
		}
		result.Items = append(result.Items, item)
		result.TotalSize = size
		result.TotalCount = count
	}

	return result
}

func (s *Scanner) scanTempFiles() cleaner.ScanResult {
	result := cleaner.ScanResult{Category: cleaner.CategoryTempFiles}

	tempPaths := []string{"/tmp", "/var/tmp"}

	for _, path := range tempPaths {
		items, size := s.scanDirectory(path, cleaner.CategoryTempFiles, false)
		result.Items = append(result.Items, items...)
		result.TotalSize += size
	}

	result.TotalCount = len(result.Items)
	return result
}

func (s *Scanner) scanDirectory(dirPath string, category cleaner.CleanCategory, isDangerous bool) ([]cleaner.CleanableItem, int64) {
	var items []cleaner.CleanableItem
	var totalSize int64

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return items, totalSize
	}

	for _, entry := range entries {
		path := filepath.Join(dirPath, entry.Name())

		var size int64
		var lastAccess time.Time

		if entry.IsDir() {
			size = s.getDirSize(path)
		} else {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			size = info.Size()
			lastAccess = info.ModTime()
		}

		item := cleaner.CleanableItem{
			Name:        entry.Name(),
			Path:        path,
			Size:        size,
			Category:    category,
			Description: path,
			LastAccess:  lastAccess,
			IsSelected:  false,
			IsDangerous: isDangerous,
		}

		items = append(items, item)
		totalSize += size
	}

	return items, totalSize
}

func (s *Scanner) getDirSize(path string) int64 {
	var size int64

	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size
}
