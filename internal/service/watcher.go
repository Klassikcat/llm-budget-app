package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"llm-budget-tracker/internal/ports"
)

type FileWatchOp uint32

const (
	FileWatchCreate FileWatchOp = 1 << iota
	FileWatchWrite
	FileWatchRemove
	FileWatchRename
	FileWatchChmod
)

type FileWatchEvent struct {
	Name string
	Op   FileWatchOp
}

type FileWatcher interface {
	Add(name string) error
	Close() error
	Events() <-chan FileWatchEvent
	Errors() <-chan error
}

type WatchReadMode int

const (
	WatchReadModeIncrementalTail WatchReadMode = iota
	WatchReadModeWholeFile
)

type WatchTarget struct {
	ID            string
	RootPath      string
	Parser        ports.SessionParser
	ReadMode      WatchReadMode
	DiscoverPaths func(root string) ([]string, error)
	ResolvePaths  func(root, changedPath string, event FileWatchEvent) []string
	MatchesPath   func(path string) bool
}

type WatchCoordinator struct {
	normalizer  *SessionNormalizerService
	checkpoints ports.CheckpointRepository
	watcher     FileWatcher
	targets     []WatchTarget

	errCh chan error

	mu       sync.Mutex
	warnings []string
	started  bool
	closed   bool
}

func NewWatchCoordinator(normalizer *SessionNormalizerService, checkpoints ports.CheckpointRepository, watcher FileWatcher, targets []WatchTarget) (*WatchCoordinator, error) {
	if normalizer == nil {
		return nil, fmt.Errorf("watch coordinator requires a session normalizer")
	}
	if checkpoints == nil {
		return nil, errCheckpointRepositoryRequired
	}
	if watcher == nil {
		return nil, errFileWatcherRequired
	}

	filtered := make([]WatchTarget, 0, len(targets))
	for _, target := range targets {
		if strings.TrimSpace(target.ID) == "" || strings.TrimSpace(target.RootPath) == "" || target.Parser == nil {
			continue
		}
		if target.ReadMode != WatchReadModeWholeFile {
			target.ReadMode = WatchReadModeIncrementalTail
		}
		filtered = append(filtered, target)
	}

	return &WatchCoordinator{
		normalizer:  normalizer,
		checkpoints: checkpoints,
		watcher:     watcher,
		targets:     filtered,
		errCh:       make(chan error, 16),
		warnings:    make([]string, 0, 8),
	}, nil
}

func (c *WatchCoordinator) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	c.started = true
	c.mu.Unlock()

	for _, target := range c.targets {
		if err := c.bootstrapTarget(ctx, target); err != nil {
			return err
		}
	}

	go c.run(ctx)
	return nil
}

func (c *WatchCoordinator) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()
	return c.watcher.Close()
}

func (c *WatchCoordinator) Errors() <-chan error { return c.errCh }

func (c *WatchCoordinator) Warnings() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	clone := make([]string, len(c.warnings))
	copy(clone, c.warnings)
	return clone
}

func (c *WatchCoordinator) bootstrapTarget(ctx context.Context, target WatchTarget) error {
	root := filepath.Clean(target.RootPath)
	info, err := os.Stat(root)
	if err != nil {
		c.recordWarning(fmt.Sprintf("watch target %s unavailable at %s: %v", target.ID, root, err))
		return nil
	}

	watchRoot := root
	if !info.IsDir() {
		watchRoot = filepath.Dir(root)
	}
	if err := c.addWatchRecursive(watchRoot); err != nil {
		c.recordWarning(fmt.Sprintf("watch target %s could not subscribe to %s: %v", target.ID, watchRoot, err))
	}

	paths, err := c.discoverPaths(target, root)
	if err != nil {
		c.recordWarning(fmt.Sprintf("watch target %s discovery failed for %s: %v", target.ID, root, err))
		return nil
	}

	for _, path := range paths {
		if err := c.ingestPath(ctx, target, path); err != nil {
			return err
		}
	}

	return nil
}

func (c *WatchCoordinator) run(ctx context.Context) {
	defer close(c.errCh)
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-c.watcher.Errors():
			if !ok {
				return
			}
			c.recordWarning(fmt.Sprintf("file watcher error: %v", err))
		case event, ok := <-c.watcher.Events():
			if !ok {
				return
			}
			if err := c.handleEvent(ctx, event); err != nil {
				select {
				case c.errCh <- err:
				default:
					c.recordWarning(fmt.Sprintf("dropping watcher error because the error channel is full: %v", err))
				}
			}
		}
	}
}

func (c *WatchCoordinator) handleEvent(ctx context.Context, event FileWatchEvent) error {
	changedPath := filepath.Clean(event.Name)
	for _, target := range c.targets {
		if !isWithinRoot(filepath.Clean(target.RootPath), changedPath) {
			continue
		}

		if event.Op&FileWatchCreate != 0 {
			if info, err := os.Stat(changedPath); err == nil && info.IsDir() {
				if err := c.addWatchRecursive(changedPath); err != nil {
					c.recordWarning(fmt.Sprintf("failed to add watch for %s: %v", changedPath, err))
				}
			}
		}

		paths := c.resolvePaths(target, changedPath, event)
		for _, path := range paths {
			if err := c.ingestPath(ctx, target, path); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *WatchCoordinator) ingestPath(ctx context.Context, target WatchTarget, sourcePath string) error {
	sourcePath = filepath.Clean(sourcePath)
	checkpointID := checkpointSourceID(target.ID, sourcePath)
	checkpoint, err := c.checkpoints.LoadCheckpoint(ctx, checkpointID)
	if err != nil {
		return fmt.Errorf("load checkpoint for %s: %w", checkpointID, err)
	}

	content, identity, nextOffsetBase, replayFromStart, err := c.readForParse(target, sourcePath, checkpoint)
	if err != nil {
		if os.IsNotExist(err) {
			c.recordWarning(fmt.Sprintf("watched source disappeared before ingest: %s", sourcePath))
			return nil
		}
		return fmt.Errorf("read watched source %s: %w", sourcePath, err)
	}

	result, err := target.Parser.Parse(ctx, ports.ParseInput{
		SourceID:   checkpointID,
		Path:       sourcePath,
		Content:    content,
		ObservedAt: time.Now().UTC(),
	})
	if err != nil {
		c.recordWarning(fmt.Sprintf("watch target %s parse warning for %s: %v", target.ID, sourcePath, err))
		return nil
	}
	for _, warning := range result.Warnings {
		c.recordWarning(fmt.Sprintf("watch target %s %s: %s", target.ID, sourcePath, warning))
	}

	if target.ReadMode == WatchReadModeIncrementalTail && !replayFromStart && shouldReplayWholeFile(content, result) {
		fullContent, fullIdentity, _, err := readWholeSource(sourcePath)
		if err != nil {
			return fmt.Errorf("re-read watched source %s after truncation fallback: %w", sourcePath, err)
		}
		content = fullContent
		identity = fullIdentity
		replayFromStart = true

		result, err = target.Parser.Parse(ctx, ports.ParseInput{
			SourceID:   checkpointID,
			Path:       sourcePath,
			Content:    content,
			ObservedAt: time.Now().UTC(),
		})
		if err != nil {
			c.recordWarning(fmt.Sprintf("watch target %s parse warning for %s after truncation fallback: %v", target.ID, sourcePath, err))
			return nil
		}
		for _, warning := range result.Warnings {
			c.recordWarning(fmt.Sprintf("watch target %s %s after truncation fallback: %s", target.ID, sourcePath, warning))
		}
	}

	events := result.Events
	if replayFromStart && checkpoint.LastMarker != "" {
		events = suppressProcessedEvents(events, checkpoint.LastMarker)
	}
	if len(events) > 0 {
		if _, err := c.normalizer.Normalize(ctx, events); err != nil {
			return fmt.Errorf("normalize watched events for %s: %w", sourcePath, err)
		}
	}

	newOffset := nextOffsetBase
	if target.ReadMode == WatchReadModeIncrementalTail && !replayFromStart {
		newOffset = nextOffsetBase + result.NextOffset
	}
	if target.ReadMode == WatchReadModeIncrementalTail && replayFromStart {
		newOffset = result.NextOffset
	}

	lastMarker := checkpoint.LastMarker
	if len(result.Events) > 0 {
		lastMarker = eventMarker(result.Events[len(result.Events)-1])
	}

	return c.checkpoints.SaveCheckpoint(ctx, ports.IngestionCheckpoint{
		SourceID:     checkpointID,
		Path:         sourcePath,
		FileIdentity: identity,
		LastMarker:   lastMarker,
		Offset:       newOffset,
		UpdatedAt:    time.Now().UTC(),
	})
}

func (c *WatchCoordinator) readForParse(target WatchTarget, sourcePath string, checkpoint ports.IngestionCheckpoint) ([]byte, string, int64, bool, error) {
	if target.ReadMode == WatchReadModeWholeFile {
		content, identity, offset, err := readWholeSource(sourcePath)
		return content, identity, offset, true, err
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, "", 0, false, err
	}
	identity := statIdentity(info)
	replayFromStart := checkpoint.Path != sourcePath || checkpoint.FileIdentity == "" || checkpoint.FileIdentity != identity || checkpoint.Offset < 0 || info.Size() < checkpoint.Offset
	if replayFromStart {
		content, err := os.ReadFile(sourcePath)
		return content, identity, 0, true, err
	}

	content, err := readFileTail(sourcePath, checkpoint.Offset)
	return content, identity, checkpoint.Offset, false, err
}

func (c *WatchCoordinator) discoverPaths(target WatchTarget, root string) ([]string, error) {
	if target.DiscoverPaths != nil {
		paths, err := target.DiscoverPaths(root)
		if err != nil {
			return nil, err
		}
		return uniqueSortedPaths(paths), nil
	}
	matcher := target.MatchesPath
	if matcher == nil {
		matcher = func(path string) bool { return true }
	}

	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if matcher(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return uniqueSortedPaths(paths), nil
}

func (c *WatchCoordinator) resolvePaths(target WatchTarget, changedPath string, event FileWatchEvent) []string {
	if target.ResolvePaths != nil {
		return uniqueSortedPaths(target.ResolvePaths(filepath.Clean(target.RootPath), changedPath, event))
	}
	if target.MatchesPath == nil || target.MatchesPath(changedPath) {
		return []string{changedPath}
	}
	return nil
}

func (c *WatchCoordinator) addWatchRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		return c.watcher.Add(path)
	})
}

func (c *WatchCoordinator) recordWarning(message string) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.warnings = append(c.warnings, trimmed)
}

func NewClaudeWatchTarget(root string, parser ports.SessionParser) WatchTarget {
	return WatchTarget{ID: parser.ParserName(), RootPath: root, Parser: parser, ReadMode: WatchReadModeIncrementalTail, MatchesPath: func(path string) bool {
		return strings.EqualFold(filepath.Ext(path), ".jsonl")
	}}
}

func NewCodexWatchTarget(root string, parser ports.SessionParser) WatchTarget {
	return WatchTarget{ID: parser.ParserName(), RootPath: root, Parser: parser, ReadMode: WatchReadModeIncrementalTail, MatchesPath: func(path string) bool {
		return strings.EqualFold(filepath.Ext(path), ".jsonl")
	}}
}

func NewGeminiWatchTarget(root string, parser ports.SessionParser) WatchTarget {
	return WatchTarget{ID: parser.ParserName(), RootPath: root, Parser: parser, ReadMode: WatchReadModeWholeFile, MatchesPath: func(path string) bool {
		base := filepath.Base(path)
		return strings.HasPrefix(base, "session-") && strings.EqualFold(filepath.Ext(base), ".json")
	}}
}

func NewOpenCodeWatchTarget(root string, parser ports.SessionParser) WatchTarget {
	return WatchTarget{
		ID:       parser.ParserName(),
		RootPath: root,
		Parser:   parser,
		ReadMode: WatchReadModeWholeFile,
		DiscoverPaths: func(root string) ([]string, error) {
			if _, err := os.Stat(root); err != nil {
				return nil, err
			}
			return []string{root}, nil
		},
		ResolvePaths: func(root, changedPath string, event FileWatchEvent) []string {
			base := filepath.Base(changedPath)
			if changedPath == root || base == "opencode.db" || base == "auth.json" {
				return []string{root}
			}
			return nil
		},
	}
}

func checkpointSourceID(targetID, sourcePath string) string {
	return strings.TrimSpace(targetID) + ":" + filepath.Clean(sourcePath)
}

func suppressProcessedEvents(events []ports.SessionEvent, lastMarker string) []ports.SessionEvent {
	lastIndex := -1
	for index, event := range events {
		if eventMarker(event) == lastMarker {
			lastIndex = index
		}
	}
	if lastIndex < 0 {
		return events
	}
	return events[lastIndex+1:]
}

func eventMarker(event ports.SessionEvent) string {
	if trimmed := strings.TrimSpace(event.ExternalID); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(event.EntryID)
}

func shouldReplayWholeFile(content []byte, result ports.ParseResult) bool {
	return len(content) > 0 && len(result.Events) == 0
}

func uniqueSortedPaths(paths []string) []string {
	set := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		set[filepath.Clean(trimmed)] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for path := range set {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func readWholeSource(path string) ([]byte, string, int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", 0, err
	}
	if info.IsDir() {
		return nil, statIdentity(info), 0, nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, "", 0, err
	}
	return content, statIdentity(info), info.Size(), nil
}

func readFileTail(path string, offset int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	return io.ReadAll(file)
}

func statIdentity(info os.FileInfo) string {
	if info == nil {
		return ""
	}
	dev, ino := statDeviceAndInode(info.Sys())
	if dev != "" || ino != "" {
		return dev + ":" + ino
	}
	return fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UTC().UnixNano())
}

func statDeviceAndInode(raw any) (string, string) {
	if raw == nil {
		return "", ""
	}
	value := reflect.ValueOf(raw)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return "", ""
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return "", ""
	}
	return fieldString(value, "Dev"), fieldString(value, "Ino")
}

func fieldString(value reflect.Value, name string) string {
	field := value.FieldByName(name)
	if !field.IsValid() {
		return ""
	}
	return fmt.Sprintf("%v", field.Interface())
}

func isWithinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && rel != "")
}
