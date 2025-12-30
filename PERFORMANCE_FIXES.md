# Performance Fixes Summary

## Problems Fixed

### 1. 10+ second blank screen on Ctrl+Q detach
**Root Cause**: Multiple blocking `CapturePane()` calls in sequence
- `processStatusUpdate()` updates 5+ sessions per tick
- Each `UpdateStatus()` → `GetStatus()` → `CapturePane()` (50-200ms)
- `AcknowledgeWithSnapshot()` → another `CapturePane()` (50-200ms)
- Total: 15 calls × 150ms = 2250ms (2.25s) per tick
- With 10 sessions and slower tmux: 10+ seconds

**Fixes Applied**:
1. **Removed `AcknowledgeWithSnapshot()` CapturePane call** (tmux.go:971-1007)
   - Use existing state tracker data instead
   - Eliminates 50-200ms blocking per attach/detach

2. **Made `AcknowledgeWithSnapshot()` async** (home.go:2612-2615)
   - Run in background goroutine
   - Don't block UI thread

3. **Skipped full status update on attach return** (home.go:1310-1316)
   - Don't trigger `processStatusUpdate()` after Ctrl+Q
   - Background worker already handles updates every tick

4. **Thread-safe `isAttaching` flag** (home.go:119, 1785, 2822, 2601, 1305)
   - Changed from `bool` to `atomic.Bool`
   - Prevents race conditions with View()

### 2. Occasional sluggishness with up/down keys
**Root Cause**: Excessive status updates and View() overhead

**Fixes Applied**:
1. **Reduced status update frequency** (home.go:45)
   - Changed tick interval: 500ms → 1s
   - Reduces CapturePane() calls by 50%

2. **Adaptive status updates** (home.go:163, 1431-1444, 1378-1388)
   - Only update statuses when user is actively interacting
   - Skip updates during idle periods (2s window)
   - Dramatically reduces overhead when user pauses

3. **Reduced batch size** (home.go:941)
   - Changed: 5 → 2 non-visible sessions per tick
   - Fewer CapturePane() calls per tick

4. **Increased cache duration** (home.go:2664)
   - Changed: 100ms → 500ms
   - Reduces `countSessionStatuses()` O(n) iterations

5. **Performance monitoring** (tmux.go:839-846)
   - Log slow CapturePane() calls (>100ms)
   - Helps identify performance bottlenecks

## Expected Impact

### For 10 sessions:
- **Before fix**: 10-15 CapturePane() calls per 500ms tick
  - 20-30 calls/sec × 100ms avg = 2-3s blocking per second
- **After fix**:
  - With user activity: 5 calls per 1s tick × 100ms = 500ms blocking per second
  - Without user activity: 0 calls (adaptive updates)

### Blank screen fix:
- **Before**: 10+ seconds (multiple sequential CapturePane() calls)
- **After**: <1 second (CapturePane() removed from hot path)

## How to Test

1. Run with debug logging:
   ```bash
   export AGENTDECK_DEBUG=1
   agent-deck
   ```

2. Check logs for slow operations:
   ```bash
   tail -f ~/.agent-deck/debug.log | grep "SLOW"
   ```

3. Test Ctrl+Q detach:
   - Should return to panel in <1 second
   - No blank screen or flicker

4. Test up/down navigation:
   - Should be responsive even with 10+ sessions
   - No noticeable lag

## Configuration Tuning

If still seeing lag, adjust these constants:

### In `internal/ui/home.go`:
```go
// Aggressive optimization (minimal status updates)
tickInterval = 2 * time.Second           // Line 48
const cacheDuration = 1 * time.Second         // Line 2664
const batchSize = 1                             // Line 941
const userActivityWindow = 5 * time.Second   // Line 1379
```

### In `internal/tmux/tmux.go`:
```go
// Increase log threshold to see all CapturePane() calls
if elapsed > 50*time.Millisecond {  // Line 844
    debugLog("...")
}
```
