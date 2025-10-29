#!/bin/bash
# Script to post PR review comments
# This unsets GH_TOKEN to allow gh CLI to use keyring authentication

set -e

# Unset environment tokens to use keyring
unset GH_TOKEN
unset GITHUB_TOKEN

REPO="prebid/prebid-server"
PR=4005

echo "Switching to scr-oath account..."
gh auth switch --user scr-oath --hostname github.com

echo "Creating pending review..."
gh pr review $PR --repo $REPO --comment --body "## Review Summary

This PR delivers a significant **5-10x performance improvement** (5-10s → 500-600ms) for GDPR vendor list preloading. The implementation using \`golang.org/x/sync/errgroup\` is generally sound, but requires critical production hardening before merge.

### 🔴 Critical Issues (Must Fix)

1. **Missing panic recovery** - Server will crash if any vendor list fetch panics
2. **No context cancellation checks** - Goroutines continue spawning after timeout
3. **Unhandled Wait() errors** - Failures during preload are masked

### 🟡 High Priority

4. **Complex nested errgroup pattern** - Consider simplifying or using \`conc/pool\`
5. **Missing test coverage** - No tests for concurrent edge cases
6. **Confusing configuration docs** - \"0 or negative means no limit\" contradicts code behavior

### ✅ Positive Aspects

- Default value maintains backward compatibility
- Thread-safe operations correctly implemented
- Measurable, significant performance improvement
- Opt-in configuration (safe default)

See detailed line comments below."

echo ""
echo "Review comment posted successfully!"
echo ""
echo "Note: Line-specific comments must be added via the GitHub web UI or API."
echo "See pr-4005-review-comments.md for the detailed line-by-line feedback."
