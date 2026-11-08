package cleaner

import "time"

// CleanCategory represents different categories of cleanable items
type CleanCategory string

const (
	CategoryPackageCache   CleanCategory = "Package Cache"
	CategoryOrphanPackages CleanCategory = "Orphan Packages"
	CategorySystemCache    CleanCategory = "System Cache"
	CategoryUserCache      CleanCategory = "User Cache"
	CategoryLogs           CleanCategory = "System Logs"
	CategoryThumbnails     CleanCategory = "Thumbnails"
	CategoryTrash          CleanCategory = "Trash"
	CategoryTempFiles      CleanCategory = "Temporary Files"
)

// CleanableItem represents a single item that can be cleaned
type CleanableItem struct {
	Name        string
	Path        string
	Size        int64
	Category    CleanCategory
	Description string
	LastAccess  time.Time
	IsSelected  bool
	IsDangerous bool // Items that might affect system stability
}

// CleanResult represents the result of a cleaning operation
type CleanResult struct {
	Category     CleanCategory
	ItemsCleaned int
	BytesFreed   int64
	Errors       []error
	Success      bool
}

// ScanResult represents the result of scanning for cleanable items
type ScanResult struct {
	Category   CleanCategory
	Items      []CleanableItem
	TotalSize  int64
	TotalCount int
	Error      error
}

// CleanOptions configures cleaning behavior
type CleanOptions struct {
	DryRun           bool
	KeepLastN        int // Keep last N versions (for package cache)
	IncludeDangerous bool
	Verbose          bool
}

// CategoryInfo provides metadata about a cleaning category
type CategoryInfo struct {
	Category     CleanCategory
	Name         string
	Description  string
	Icon         string
	IsDangerous  bool
	RequiresRoot bool
}

// GetCategoryInfo returns information about all available categories
func GetCategoryInfo() []CategoryInfo {
	return []CategoryInfo{
		{
			Category:     CategoryPackageCache,
			Name:         "Package Cache",
			Description:  "Cached package files from pacman downloads",
			Icon:         "\uf187 ", // nf-fa-archive
			IsDangerous:  false,
			RequiresRoot: true,
		},
		{
			Category:     CategoryOrphanPackages,
			Name:         "Orphan Packages",
			Description:  "Packages installed as dependencies but no longer required",
			Icon:         "\uf6bb ", // nf-fa-ghost
			IsDangerous:  true,
			RequiresRoot: true,
		},
		{
			Category:     CategorySystemCache,
			Name:         "System Cache",
			Description:  "System-wide cache files",
			Icon:         "\uf1c0 ", // nf-fa-database
			IsDangerous:  false,
			RequiresRoot: true,
		},
		{
			Category:     CategoryUserCache,
			Name:         "User Cache",
			Description:  "User application cache files (~/.cache)",
			Icon:         "\uf0c7 ", // nf-fa-save
			IsDangerous:  false,
			RequiresRoot: false,
		},
		{
			Category:     CategoryLogs,
			Name:         "System Logs",
			Description:  "Old system and journal logs",
			Icon:         "\uf15c ", // nf-fa-file-text
			IsDangerous:  false,
			RequiresRoot: true,
		},
		{
			Category:     CategoryThumbnails,
			Name:         "Thumbnails",
			Description:  "Cached thumbnail images",
			Icon:         "\uf03e ", // nf-fa-image
			IsDangerous:  false,
			RequiresRoot: false,
		},
		{
			Category:     CategoryTrash,
			Name:         "Trash",
			Description:  "Files in trash/recycle bin",
			Icon:         "\uf1f8 ", // nf-fa-trash
			IsDangerous:  false,
			RequiresRoot: false,
		},
		{
			Category:     CategoryTempFiles,
			Name:         "Temporary Files",
			Description:  "Temporary files in /tmp and /var/tmp",
			Icon:         "\uf252 ", // nf-fa-hourglass-half
			IsDangerous:  false,
			RequiresRoot: true,
		},
	}
}
