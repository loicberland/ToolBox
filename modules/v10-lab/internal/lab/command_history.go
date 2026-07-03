package lab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"toolBox/pkg/toolboxruntime"
)

const commandHistoryVersion = 1
const commandHistoryRecentLimit = 30

var commandHistoryMu sync.Mutex
var commandHistoryNow = time.Now

type ExecutableCommandHistoryEntry struct {
	ID             string                      `json:"id"`
	TargetKind     ExecutableCommandTargetKind `json:"targetKind"`
	TargetName     string                      `json:"targetName"`
	Command        string                      `json:"command"`
	LastExecutedAt string                      `json:"lastExecutedAt"`
	ExecutionCount int                         `json:"executionCount"`
	Favorite       bool                        `json:"favorite"`
}

type CommandHistoryRegistry struct {
	Version   int                                        `json:"version"`
	Maquettes map[string][]ExecutableCommandHistoryEntry `json:"maquettes"`
}

func CommandHistoryPath() string {
	layout, err := toolboxruntime.ForModule(ModuleID)
	if err != nil {
		return filepath.Join("data", ModuleID, "command-history.json")
	}
	return filepath.Join(layout.DataDir, "command-history.json")
}

func ListExecutableCommandHistory(maquetteName string) ([]ExecutableCommandHistoryEntry, error) {
	commandHistoryMu.Lock()
	defer commandHistoryMu.Unlock()
	registry, err := loadCommandHistoryRegistry()
	if err != nil {
		return nil, err
	}
	return cloneCommandHistoryEntries(registry.Maquettes[commandHistoryMaquetteKey(maquetteName)]), nil
}

func RecordExecutableCommand(maquetteName string, entry ExecutableCommandHistoryEntry) ([]ExecutableCommandHistoryEntry, error) {
	commandHistoryMu.Lock()
	defer commandHistoryMu.Unlock()
	entry.TargetKind = ExecutableCommandTargetKind(strings.TrimSpace(string(entry.TargetKind)))
	entry.TargetName = strings.TrimSpace(entry.TargetName)
	if err := validateExecutableCommandHistoryEntry(entry); err != nil {
		return nil, err
	}
	registry, err := loadCommandHistoryRegistry()
	if err != nil {
		return nil, err
	}
	key := commandHistoryMaquetteKey(maquetteName)
	now := commandHistoryNow().Format(time.RFC3339)
	items := cloneCommandHistoryEntries(registry.Maquettes[key])
	next := ExecutableCommandHistoryEntry{
		ID:             commandHistoryEntryID(entry, now, items),
		TargetKind:     entry.TargetKind,
		TargetName:     entry.TargetName,
		Command:        entry.Command,
		LastExecutedAt: now,
		ExecutionCount: 1,
		Favorite:       entry.Favorite,
	}
	for index, item := range items {
		if sameExecutableCommandHistoryEntry(item, entry) {
			next = item
			next.LastExecutedAt = now
			next.ExecutionCount++
			items = append(items[:index], items[index+1:]...)
			break
		}
	}
	items = append([]ExecutableCommandHistoryEntry{next}, items...)
	registry.Maquettes[key] = pruneExecutableCommandHistory(items)
	if err := saveCommandHistoryRegistry(registry); err != nil {
		return nil, err
	}
	return cloneCommandHistoryEntries(registry.Maquettes[key]), nil
}

func SetExecutableCommandHistoryFavorite(maquetteName string, id string, favorite bool) ([]ExecutableCommandHistoryEntry, error) {
	commandHistoryMu.Lock()
	defer commandHistoryMu.Unlock()
	registry, err := loadCommandHistoryRegistry()
	if err != nil {
		return nil, err
	}
	key := commandHistoryMaquetteKey(maquetteName)
	items := cloneCommandHistoryEntries(registry.Maquettes[key])
	found := false
	for index := range items {
		if items[index].ID == strings.TrimSpace(id) {
			items[index].Favorite = favorite
			found = true
			break
		}
	}
	if !found {
		return nil, os.ErrNotExist
	}
	registry.Maquettes[key] = items
	if err := saveCommandHistoryRegistry(registry); err != nil {
		return nil, err
	}
	return cloneCommandHistoryEntries(items), nil
}

func DeleteExecutableCommandHistoryEntry(maquetteName string, id string) ([]ExecutableCommandHistoryEntry, error) {
	commandHistoryMu.Lock()
	defer commandHistoryMu.Unlock()
	registry, err := loadCommandHistoryRegistry()
	if err != nil {
		return nil, err
	}
	key := commandHistoryMaquetteKey(maquetteName)
	items := cloneCommandHistoryEntries(registry.Maquettes[key])
	next := items[:0]
	found := false
	for _, item := range items {
		if item.ID == strings.TrimSpace(id) {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		return nil, os.ErrNotExist
	}
	registry.Maquettes[key] = next
	if err := saveCommandHistoryRegistry(registry); err != nil {
		return nil, err
	}
	return cloneCommandHistoryEntries(next), nil
}

func ClearExecutableCommandHistoryNonFavorites(maquetteName string) ([]ExecutableCommandHistoryEntry, error) {
	commandHistoryMu.Lock()
	defer commandHistoryMu.Unlock()
	registry, err := loadCommandHistoryRegistry()
	if err != nil {
		return nil, err
	}
	key := commandHistoryMaquetteKey(maquetteName)
	items := cloneCommandHistoryEntries(registry.Maquettes[key])
	next := items[:0]
	for _, item := range items {
		if item.Favorite {
			next = append(next, item)
		}
	}
	registry.Maquettes[key] = next
	if err := saveCommandHistoryRegistry(registry); err != nil {
		return nil, err
	}
	return cloneCommandHistoryEntries(next), nil
}

func RenameCommandHistoryMaquette(oldName string, newName string) error {
	oldKey := commandHistoryMaquetteKey(oldName)
	newKey := commandHistoryMaquetteKey(newName)
	if oldKey == newKey {
		return nil
	}
	commandHistoryMu.Lock()
	defer commandHistoryMu.Unlock()
	registry, err := loadCommandHistoryRegistry()
	if err != nil {
		return err
	}
	items, ok := registry.Maquettes[oldKey]
	if !ok {
		return nil
	}
	if _, exists := registry.Maquettes[newKey]; exists {
		return fmt.Errorf("historique de commandes deja present pour %s", newName)
	}
	registry.Maquettes[newKey] = items
	delete(registry.Maquettes, oldKey)
	return saveCommandHistoryRegistry(registry)
}

func validateExecutableCommandHistoryEntry(entry ExecutableCommandHistoryEntry) error {
	if entry.TargetKind == "" {
		return fmt.Errorf("type de cible requis")
	}
	if entry.TargetName == "" {
		return fmt.Errorf("cible requise")
	}
	if strings.TrimSpace(entry.Command) == "" {
		return fmt.Errorf("commande executable requise")
	}
	if _, err := splitCommandLine(strings.TrimSpace(entry.Command)); err != nil {
		return err
	}
	switch entry.TargetKind {
	case ExecutableCommandTargetRoot, ExecutableCommandTargetService, ExecutableCommandTargetConnector, ExecutableCommandTargetAgent, ExecutableCommandTargetAdaptor:
		return nil
	default:
		return fmt.Errorf("type de cible inconnu: %s", entry.TargetKind)
	}
}

func loadCommandHistoryRegistry() (CommandHistoryRegistry, error) {
	path := CommandHistoryPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return CommandHistoryRegistry{Version: commandHistoryVersion, Maquettes: map[string][]ExecutableCommandHistoryEntry{}}, nil
	}
	if err != nil {
		return CommandHistoryRegistry{}, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return CommandHistoryRegistry{Version: commandHistoryVersion, Maquettes: map[string][]ExecutableCommandHistoryEntry{}}, nil
	}
	var registry CommandHistoryRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return CommandHistoryRegistry{}, fmt.Errorf("fichier d'historique des commandes invalide: %w", err)
	}
	if registry.Version == 0 {
		registry.Version = commandHistoryVersion
	}
	if registry.Maquettes == nil {
		registry.Maquettes = map[string][]ExecutableCommandHistoryEntry{}
	}
	for key, items := range registry.Maquettes {
		registry.Maquettes[key] = normalizeExecutableCommandHistory(items)
	}
	return registry, nil
}

func saveCommandHistoryRegistry(registry CommandHistoryRegistry) error {
	path := CommandHistoryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	registry.Version = commandHistoryVersion
	if registry.Maquettes == nil {
		registry.Maquettes = map[string][]ExecutableCommandHistoryEntry{}
	}
	payload, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), "command-history-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	if _, err := temp.Write(append(payload, '\n')); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempName)
		return err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempName)
		return err
	}
	_ = os.Remove(path)
	if err := os.Rename(tempName, path); err != nil {
		_ = os.Remove(tempName)
		return err
	}
	return nil
}

func normalizeExecutableCommandHistory(items []ExecutableCommandHistoryEntry) []ExecutableCommandHistoryEntry {
	next := []ExecutableCommandHistoryEntry{}
	for _, item := range items {
		if item.ID == "" || item.ExecutionCount <= 0 || item.LastExecutedAt == "" {
			continue
		}
		if strings.TrimSpace(item.Command) == "" || strings.TrimSpace(item.TargetName) == "" || strings.TrimSpace(string(item.TargetKind)) == "" {
			continue
		}
		item.TargetName = strings.TrimSpace(item.TargetName)
		item.TargetKind = ExecutableCommandTargetKind(strings.TrimSpace(string(item.TargetKind)))
		next = append(next, item)
	}
	sort.SliceStable(next, func(i, j int) bool {
		return next[i].LastExecutedAt > next[j].LastExecutedAt
	})
	return pruneExecutableCommandHistory(next)
}

func pruneExecutableCommandHistory(items []ExecutableCommandHistoryEntry) []ExecutableCommandHistoryEntry {
	next := []ExecutableCommandHistoryEntry{}
	nonFavoriteCount := 0
	for _, item := range items {
		if !item.Favorite {
			nonFavoriteCount++
			if nonFavoriteCount > commandHistoryRecentLimit {
				continue
			}
		}
		next = append(next, item)
	}
	return next
}

func sameExecutableCommandHistoryEntry(left ExecutableCommandHistoryEntry, right ExecutableCommandHistoryEntry) bool {
	return left.TargetKind == right.TargetKind &&
		strings.EqualFold(strings.TrimSpace(left.TargetName), strings.TrimSpace(right.TargetName)) &&
		strings.TrimSpace(left.Command) == strings.TrimSpace(right.Command)
}

func commandHistoryMaquetteKey(name string) string {
	return strings.ToLower(safeDirName(name))
}

func commandHistoryEntryID(entry ExecutableCommandHistoryEntry, timestamp string, existing []ExecutableCommandHistoryEntry) string {
	base := strings.ToLower(safeDirName(fmt.Sprintf("%s-%s-%s", entry.TargetKind, entry.TargetName, strings.TrimSpace(entry.Command))))
	if base == "" || base == "sans-nom" {
		base = "commande"
	}
	base = fmt.Sprintf("%s-%s", base, strings.NewReplacer(":", "", "-", "", "T", "-", "Z", "").Replace(timestamp))
	used := map[string]bool{}
	for _, item := range existing {
		used[item.ID] = true
	}
	if !used[base] {
		return base
	}
	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s-%d", base, index)
		if !used[candidate] {
			return candidate
		}
	}
}

func cloneCommandHistoryEntries(items []ExecutableCommandHistoryEntry) []ExecutableCommandHistoryEntry {
	if items == nil {
		return []ExecutableCommandHistoryEntry{}
	}
	return append([]ExecutableCommandHistoryEntry{}, items...)
}
