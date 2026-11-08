package components

import (
	"fmt"
	"strings"

	"github.com/Gylmynnn/goclean/internal/cleaner"
	"github.com/Gylmynnn/goclean/internal/tui/styles"
)

type HelpKey struct {
	Key  string
	Desc string
}

func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func FormatSizeAligned(bytes int64) string {
	size := FormatSize(bytes)
	return fmt.Sprintf("%9s", size)
}

func RenderHelp(keys []HelpKey) string {
	var parts []string
	for _, hk := range keys {
		parts = append(parts, fmt.Sprintf("%s %s",
			styles.HelpKeyStyle.Render(hk.Key),
			hk.Desc,
		))
	}
	return styles.HelpStyle.Render(strings.Join(parts, "  •  "))
}

func RenderResult(result cleaner.CleanResult) string {
	var status string
	if result.Success {
		status = styles.SuccessStyle.Render("\uf00c ") // nf-fa-check
	} else {
		status = styles.ErrorStyle.Render("\uf00d ") // nf-fa-times
	}

	var info strings.Builder
	fmt.Fprintf(&info, "%s %s: %d items, %s freed",
		status,
		result.Category,
		result.ItemsCleaned,
		FormatSize(result.BytesFreed))

	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			info.WriteString("\n    ")
			info.WriteString(styles.ErrorStyle.Render(err.Error()))
		}
	}

	return info.String()
}
