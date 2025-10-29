#!/bin/bash

# PR 4005 Review Script
# Run this after authenticating with: gh auth login
# Make sure you're authenticated as scr-oath

set -e

REPO="prebid/prebid-server"
PR_NUM=4005

echo "Posting main review comment..."
gh pr comment $PR_NUM --repo $REPO --body "## Review Summary

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

Detailed line comments below."

echo "Main comment posted. Now you need to add line-specific review comments manually or via GitHub web UI."
echo ""
echo "Line comments to add:"
echo ""
echo "1. gdpr/vendorlist-fetching.go:92-109 - Missing panic recovery (CRITICAL)"
echo "2. gdpr/vendorlist-fetching.go:98-106 - Check context cancellation (CRITICAL)"
echo "3. gdpr/vendorlist-fetching.go:108-111 - Handle Wait() errors (CRITICAL)"
echo "4. gdpr/vendorlist-fetching.go:82-111 - Nested errgroup complexity (HIGH)"
echo "5. gdpr/vendorlist-fetching.go:112 - Add failure tracking (MEDIUM)"
echo "6. config/config.go:244-248 - Clarify config docs (HIGH)"
echo "7. gdpr/vendorlist-fetching_test.go:256-262 - Missing test coverage (HIGH)"
echo "8. gdpr/vendorlist-fetching_test.go:295-297 - Unrelated change? (MEDIUM)"
echo "9. sample/001_banner/app.yaml:9-14 - Document sample config (LOW)"
echo ""
echo "See pr-4005-review-comments.md for full comment text"
