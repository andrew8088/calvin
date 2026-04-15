package hooks

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/logging"
)

type Hook struct {
	Name string
	Type string
	Path string
}

var ValidTypes = []string{"before-event-start", "on-event-start", "on-event-end"}

func Discover() (map[string][]Hook, error) {
	return DiscoverFrom(config.HooksDir())
}

func DiscoverFrom(baseDir string) (map[string][]Hook, error) {
	result := make(map[string][]Hook)
	log := logging.Get()

	for _, hookType := range ValidTypes {
		dir := filepath.Join(baseDir, hookType)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		var hooks []Hook
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.Mode()&0111 == 0 {
				log.Warn("hooks", "Skipping non-executable hook: "+path+". Fix: chmod +x "+path)
				continue
			}

			hooks = append(hooks, Hook{
				Name: entry.Name(),
				Type: hookType,
				Path: path,
			})
		}

		sort.Slice(hooks, func(i, j int) bool {
			return hooks[i].Name < hooks[j].Name
		})

		if len(hooks) > 0 {
			result[hookType] = hooks
		}
	}

	return result, nil
}

func CountByType() (map[string]int, error) {
	hooks, err := Discover()
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for hookType, list := range hooks {
		counts[hookType] = len(list)
	}
	return counts, nil
}
