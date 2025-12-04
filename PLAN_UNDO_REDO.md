# Undo/Redo System Design

## Overview

Centralized, Go-based history system that tracks all sitemap modifications with support for undo/redo operations.

## Architecture

### Core Concepts

1. **Action** - Atomic unit of work (create node, delete node, update, etc.)
2. **Transaction** - Group of actions that should be undone/redone together (e.g., AI generation = 1 transaction with N node creates)
3. **History Stack** - Per-source (sitemap) stack of transactions
4. **History Manager** - Singleton service managing all stacks

### Components

```
┌─────────────────────────────────────────────────────────────────┐
│                         UI (React)                               │
│  - Calls handlers (CreateNode, Import, Scan, etc.)              │
│  - Calls Undo/Redo handlers                                      │
│  - Receives state updates via events                             │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                    SitemapsHandler                               │
│  - All operations go through here                                │
│  - Wraps operations in transactions                              │
│  - Exposes Undo/Redo/CanUndo/CanRedo                            │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                    HistoryManager                                │
│  - Generic, reusable for any domain                             │
│  - Manages stacks per source (sitemapId, etc.)                  │
│  - Enforces max history limit                                    │
│  - Provides Undo/Redo execution                                  │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                    Action Interface                              │
│  - Do() - execute action                                         │
│  - Undo() - reverse action                                       │
│  - Description() - human-readable name                           │
└─────────────────────────────────────────────────────────────────┘
```

## Action Types for Sitemap

| Action | Do | Undo | Data Needed |
|--------|-----|------|-------------|
| CreateNode | Insert node | Delete node | Node data |
| DeleteNode | Delete node | Re-insert node | Full node snapshot |
| UpdateNode | Update fields | Restore old fields | Old + New data |
| MoveNode | Change parent | Restore old parent | Node ID, old/new parent |
| ImportNodes | Create many | Delete all created | List of created IDs |
| ScanNodes | Create many | Delete all created | List of created IDs |
| GenerateNodes | Create many | Delete all created | List of created IDs |
| BulkCreate | Create many | Delete all created | List of created IDs |

## Implementation Plan

### Phase 1: Core History Infrastructure

1. Create `pkg/history/` package with:
   - `Action` interface
   - `Transaction` struct
   - `Stack` struct (undo/redo stacks)
   - `Manager` struct (manages multiple stacks)

### Phase 2: Sitemap Actions

2. Create `internal/domain/sitemap/actions/` with:
   - `CreateNodeAction`
   - `DeleteNodeAction`
   - `UpdateNodeAction`
   - `MoveNodeAction`
   - `BatchCreateAction` (for import/scan/generate)

### Phase 3: Integration

3. Update `SitemapsHandler`:
   - Inject HistoryManager
   - Wrap each operation in transaction
   - Add `Undo`, `Redo`, `CanUndo`, `CanRedo` methods
   - Add `GetHistoryInfo` for UI state

4. Update UI:
   - Remove local history hooks
   - Call Go handlers for undo/redo
   - Subscribe to history state changes

## API

### Handler Methods

```go
// Execute undo for specific sitemap
func (h *SitemapsHandler) Undo(sitemapId int64) *dto.Response[bool]

// Execute redo for specific sitemap
func (h *SitemapsHandler) Redo(sitemapId int64) *dto.Response[bool]

// Get history state for UI
func (h *SitemapsHandler) GetHistoryState(sitemapId int64) *dto.Response[HistoryStateDTO]

type HistoryStateDTO struct {
    CanUndo     bool   `json:"canUndo"`
    CanRedo     bool   `json:"canRedo"`
    UndoCount   int    `json:"undoCount"`
    RedoCount   int    `json:"redoCount"`
    LastAction  string `json:"lastAction,omitempty"`  // Description of last action
}
```

### Session Management

- History stack is created when editor opens (first GetSitemapWithNodes call)
- History stack is cleared when editor closes (ClearHistory call or timeout)
- Each sitemap has independent history

## Memory Considerations

- Max 25 transactions per sitemap (configurable)
- Transactions older than limit are dropped
- Stack cleared on session end (editor close)
- Node snapshots stored for delete operations (needed for undo)

## Event Flow Example

### AI Generate Flow

```
1. UI: GenerateSitemapStructure(input)
2. Handler:
   - Start transaction "AI Generate (15 nodes)"
   - Call AI service
   - For each generated node:
     - Create node in DB
     - Add CreateNodeAction to transaction
   - Commit transaction to history
3. UI: Reload data, buttons update based on CanUndo/CanRedo

4. UI: Undo()
5. Handler:
   - Pop transaction from undo stack
   - For each action in reverse:
     - Call action.Undo() (deletes node)
   - Push transaction to redo stack
6. UI: Reload data
```
