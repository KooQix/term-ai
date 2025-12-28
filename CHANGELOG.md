# TermAI Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Fixed
- **Paste Delay Issue** (2025-12-28)
  - Fixed multi-line paste delay where lines appeared sequentially with visible delay
  - Upgraded bubbletea from v0.23.2 to v0.26.6 with bracketed paste support
  - Upgraded bubbles from v0.15.0 to v0.18.0
  - Added paste event optimization to skip expensive operations during paste
  - All pasted content now appears instantly without line-by-line animation
  - See [PASTE_FIX.md](PASTE_FIX.md) for technical details

### Changed
- **Dependencies Updated**
  - `github.com/charmbracelet/bubbletea`: v0.23.2 → v0.26.6
  - `github.com/charmbracelet/bubbles`: v0.15.0 → v0.18.0
  - `github.com/charmbracelet/lipgloss`: v0.6.0 → v0.9.1

### Added
- Test script for verifying paste performance (`test_paste.sh`)
- Documentation for paste fix implementation (`PASTE_FIX.md`)

## Previous Changes
See [CHANGELOG_FIXES.md](CHANGELOG_FIXES.md) for previous bug fixes and features.
