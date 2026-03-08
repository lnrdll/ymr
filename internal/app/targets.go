package app

import (
	"log/slog"
	"slices"
)

func filterTargets(specTargetIds []string, overrideTargets []string) []string {
	if len(overrideTargets) == 0 {
		slog.Debug("Rendering all targets", "count", len(specTargetIds))
		return specTargetIds
	}

	filtered := make([]string, 0)
	for _, t := range overrideTargets {
		if slices.Contains(specTargetIds, t) {
			filtered = append(filtered, t)
		} else {
			slog.Warn("Requested target not found in spec", "targetId", t)
		}
	}

	slog.Debug("Rendering specific targets", "targets", filtered)
	return filtered
}
