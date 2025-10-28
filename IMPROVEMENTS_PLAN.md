# Hook Execution Improvements Plan

## Overview
This document outlines recommended improvements for the hook execution code in `hooks/hookexecution/execution.go` based on comprehensive code review by specialized agents (code-reviewer, golang-pro, and refactoring-specialist).

## Current State
The mutations-in-sequence branch successfully refactors the hook execution pattern from channel-based collection to pre-allocated slice approach. The changes simplify the concurrency model and improve performance.

---

## Critical Issues (Must Fix)

### 1. Uninitialized Response in Rejected Case
**Location:** `hooks/hookexecution/execution.go:146-149`

**Issue:**
When a hook receives the rejection signal (another hook has rejected), it returns without writing to `*resp`, leaving a zero-value `hookResponse` in the array.

**Current Code:**
```go
case <-rejected:
    // In this path, rejected has already been reported; no need to report it again.
    return false
```

**Problem:**
- The hookResponse at that index remains a zero value
- `handleHookResponses` will process this zero-value entry (line 162)
- The HookID will be empty (zero value)
- Could cause nil pointer dereferences or incorrect behavior downstream

**Recommended Fix:**
```go
case <-rejected:
    // Another hook has already rejected this group; record a cancellation response.
    // This ensures the response slot is properly initialized rather than left as a zero value.
    *resp = hookResponse[P]{
        HookID:        hookId,
        ExecutionTime: time.Since(startTime),
        Result:        hookstage.HookResult[P]{},
    }
    return false
```

---

### 2. Goroutine Leak on Timeout/Rejection
**Location:** `hooks/hookexecution/execution.go:115-130`

**Issue:**
Context creation is inside the child goroutine, preventing parent from canceling it when timeout or rejection occurs. The child goroutine continues executing `hookHandler` even after the parent has moved on.

**Current Code:**
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            glog.Errorf("...")
        }
    }()

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    result, err := hookHandler(ctx, moduleCtx, hw.Hook, payload)
    hookRespCh <- hookResponse[P]{
        Result: result,
        Err:    err,
    }
}()
```

**Problem:**
- When parent times out or receives rejection, child goroutine keeps running
- Wasted CPU cycles executing hooks after timeout/rejection
- Memory retained longer than necessary
- Context timeout is in child, so parent can't cancel it

**Recommended Fix:**
```go
// Create context at parent level with timeout to enable proper cancellation of child goroutine
ctx, cancel := context.WithTimeout(context.Background(), timeout)
defer cancel() // Ensures child goroutine is canceled on any return path

go func() {
    defer func() {
        if r := recover(); r != nil {
            glog.Errorf("...")
        }
    }()

    result, err := hookHandler(ctx, moduleCtx, hw.Hook, payload)
    // Use select to avoid sending on channel if context is already canceled
    select {
    case hookRespCh <- hookResponse[P]{
        Result: result,
        Err:    err,
    }:
    case <-ctx.Done():
        // Context was canceled, don't send result
    }
}()
```

**Impact:** High - In high-throughput systems with frequent timeouts or rejections, this creates significant resource waste.

---

### 3. Data Race on newPayload
**Location:** `hooks/hookexecution/execution.go:85-93`

**Issue:**
The variable `newPayload` is captured by reference in the goroutine closure. Each iteration may create a different `newPayload`, but goroutines might use the wrong value.

**Current Code:**
```go
for i, hook := range group.Hooks {
    mCtx := executionCtx.getModuleContext(hook.Module)
    mCtx.HookImplCode = hook.Code
    newPayload := handleModuleActivities(hook.Code, executionCtx.activityControl, payload, executionCtx.account)
    wg.Add(1)
    go func(hw hooks.HookWrapper[H], hookResp *hookResponse[P], moduleCtx hookstage.ModuleInvocationContext) {
        defer wg.Done()
        if executeHook(moduleCtx, hw, newPayload, hookHandler, group.Timeout, hookResp, rejected) {
            // ...
        }
    }(hook, &hookResponses[i], mCtx)
}
```

**Problem:**
- `newPayload` is captured by reference from the loop
- Goroutines may execute after the loop has moved to the next iteration
- All goroutines might use the last iteration's `newPayload`

**Recommended Fix:**
```go
go func(hw hooks.HookWrapper[H], hookResp *hookResponse[P], moduleCtx hookstage.ModuleInvocationContext, hookPayload P) {
    defer wg.Done()
    if executeHook(moduleCtx, hw, hookPayload, hookHandler, group.Timeout, hookResp, rejected) {
        closeRejectedOnce()
    }
}(hook, &hookResponses[i], mCtx, newPayload)
```

**Testing:** Run `go test -race ./hooks/hookexecution/...` to verify fix

---

## High Priority Issues

### 4. Timer Leak with time.After
**Location:** `hooks/hookexecution/execution.go:138`

**Issue:**
`time.After()` creates a timer that leaks until it fires, even if the hook completes early.

**Current Code:**
```go
case <-time.After(timeout):
```

**Problem:**
- Creates a new timer for every hook execution
- If timer fires after hook completes, it leaks until garbage collection
- In high-throughput systems, creates memory pressure

**Recommended Fix:**
```go
// Use time.NewTimer instead of time.After to avoid timer leaks
timer := time.NewTimer(timeout)
defer timer.Stop() // Ensure timer is always stopped to prevent leaks

select {
case res := <-hookRespCh:
    timer.Stop() // Stop immediately on success
    res.HookID = hookId
    res.ExecutionTime = time.Since(startTime)
    *resp = res
    return res.Result.Reject
case <-timer.C:
    *resp = hookResponse[P]{
        Err:           TimeoutError{},
        ExecutionTime: time.Since(startTime),
        HookID:        hookId,
        Result:        hookstage.HookResult[P]{},
    }
    return false
case <-rejected:
    // ...
}
```

---

### 5. Missing Zero-Value Check in handleHookResponses
**Location:** `hooks/hookexecution/execution.go:162-176`

**Issue:**
After issue #1 is fixed, `handleHookResponses` should skip or specially handle responses from hooks that were canceled early.

**Current Code:**
```go
for _, r := range hookResponses {
    groupModuleCtx[r.HookID.ModuleCode] = r.Result.ModuleContext
    // ...
}
```

**Problem:**
If `r.HookID.ModuleCode` is an empty string (zero value), this creates a map entry with an empty key.

**Recommended Fix:**
```go
for _, r := range hookResponses {
    // Skip responses from hooks that were canceled due to another hook's rejection.
    // These responses have empty HookID.ModuleCode and shouldn't be processed.
    if r.HookID.ModuleCode == "" {
        continue
    }
    groupModuleCtx[r.HookID.ModuleCode] = r.Result.ModuleContext
    // ...
}
```

---

## Medium Priority Issues

### 6. Behavioral Change Documentation
**Issue:**
The refactor changes behavior when hooks are rejected:

**OLD Behavior:**
- When a hook rejected, `collectHookResponses` would break immediately
- Only responses collected up to rejection were processed
- Subsequent hooks never finished executing

**NEW Behavior:**
- All hooks continue executing in their goroutines
- `wg.Wait()` blocks until ALL hooks complete
- All responses are collected; `handleHookResponses` stops at first rejection

**Impact:**
- Performance: Unnecessary computation continues after rejection
- Metrics: Hooks that would have been skipped now get counted
- Test expectations: Tests expecting early termination may fail

**Recommendation:**
Document this behavioral change explicitly in:
1. Commit message
2. PR description
3. Code comments
4. Release notes

---

### 7. Documentation Improvements

**Add to executeHook function:**
```go
// executeHook executes a single hook and returns the rejected status.
// The hook receives a context with timeout for proper cancellation.
// When another hook rejects (rejected channel closes), this hook is canceled.
// The resp pointer is always written to with the final hook response.
//
// Returns true if this hook rejected the request, false otherwise.
// If another hook has already rejected (rejected channel is closed), this function
// returns early without executing the hook handler.
```

**Add package-level comment:**
```go
// Package hookexecution provides concurrent execution of hooks with the following guarantees:
//
// Memory Safety:
// - Each hook execution writes to a pre-allocated slice element
// - sync.WaitGroup provides happens-before guarantees between writes and reads
// - The rejected channel is closed at most once via sync.Once
//
// Cancellation:
// - When any hook rejects, all other hooks receive a cancellation signal
// - Hooks check the rejection signal at strategic points
// - Hooks already executing will run to completion
//
// Timeout Handling:
// - Each hook has an individual timeout
// - Timed-out hooks don't prevent other hooks from completing
// - Hook handlers should respect context cancellation for proper cleanup
```

---

## Testing Requirements

### Before Merging to Production

1. **Race Detector Test:**
```bash
go test -race ./hooks/hookexecution/...
```

2. **Rejection Behavior Test:**
Create test with 5 hooks where hook #2 rejects, verify:
- Only hooks 1-2 are processed by handleHookResponses
- Hooks 3-5 still execute (but responses are ignored after rejection)
- Metrics are correctly recorded

3. **Early Exit Test:**
Verify that when `rejected` channel closes, hooks in the `<-rejected` case write valid responses

4. **Concurrent Rejection Test:**
Multiple hooks attempting to reject simultaneously should only result in one rejection being recorded

5. **Memory Profile:**
Compare memory usage before/after with large hook groups

---

## Performance Analysis

### Memory Allocation Comparison

**Before:**
- Buffered channel: O(n) where n = number of hooks
- Dynamic slice growth: O(n) with potential reallocations
- Response collection goroutine: 1 goroutine + stack overhead

**After:**
- No response channel
- Pre-allocated slice: O(n) with no reallocations
- No collection goroutine
- Saved: ~1 allocation per group

**Verdict:** Modest improvement (~1 allocation per group)

### Concurrency Characteristics

**Before:**
- Hooks execute concurrently
- Results collected via channel (serialization point)
- Rejection stops collection but doesn't signal other hooks immediately

**After:**
- Hooks execute concurrently
- Results written directly to slice (no serialization)
- Rejection signals other hooks via channel (better early termination)

**Verdict:** Improved rejection propagation, eliminated serialization bottleneck

---

## Summary of Recommendations

### Priority 1 (Critical - Must Fix Before Production):
1. ✅ Fix uninitialized response in rejected case
2. ✅ Fix goroutine leak by moving context to parent level
3. ✅ Fix data race on newPayload

### Priority 2 (High - Should Fix):
4. ✅ Replace `time.After` with `time.NewTimer` to avoid timer leaks
5. ✅ Add zero-value check in handleHookResponses

### Priority 3 (Medium - Code Quality):
6. Document behavioral change in PR description
7. Improve function and package documentation

### Testing:
- Run race detector tests
- Add concurrency-specific tests
- Verify memory profile

---

## Validation Checklist

- [ ] All critical issues (1-3) fixed
- [ ] All high priority issues (4-5) fixed
- [ ] `go test -race ./hooks/hookexecution/...` passes with zero races
- [ ] All existing tests pass
- [ ] Behavioral changes documented in PR
- [ ] Function documentation updated
- [ ] Memory profile acceptable for production use

---

## Conclusion

The refactor improves code clarity and eliminates a serialization point in response collection. The core concurrency patterns are sound (WaitGroup synchronization, sync.Once for channel closing).

However, **critical resource leak issues exist** with goroutine management and timer cleanup. These must be addressed before production deployment, especially in high-throughput environments where hooks frequently timeout or get rejected.

With the recommended fixes, this will be a solid, production-ready implementation.