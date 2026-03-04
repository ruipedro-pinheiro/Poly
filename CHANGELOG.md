# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.7.0] - 2026-03-04

### Added
- **Universal Agentic Loop**: Introduced a unified orchestration layer in `internal/llm/agent.go` that handles multi-turn tool execution for all providers.
- **Custom Provider Enhancements**: Custom providers now support the full agentic loop, receive system prompts, and are fully integrated into the UI.
- **GitHub Copilot**: Native support for GitHub Copilot with Device Flow authentication.
- **OAuth PKCE Helpers**: Centralized OAuth PKCE logic for Anthropic, OpenAI, and Google.
- **Thinking Mode**: Support for reasoning models and thinking blocks in the TUI.

### Changed
- **Architecture Refactor**: Significant reduction in code duplication (~1000 lines removed) by unifying provider implementations.
- **Theme System**: Centralized Catppuccin theme management in `internal/theme`.
- **TUI Optimization**: Improved message rendering, viewport navigation, and command palette handling.
- **Version Management**: Transitioned to an automated release process with GoReleaser.

### Fixed
- **Security Hardening**:
    - Hardened Sandbox with resource limits, capability dropping, and privilege escalation prevention.
    - Fixed Path Traversal vulnerabilities in file tools via strict symlink resolution.
    - Fixed AppleScript command injection in macOS notifications.
    - Restricted sensitive file permissions to `0600`.
- **Stability**:
    - Fixed race conditions in streaming event handling.
    - Improved HTTP client robustness with centralized retry logic and jittered backoff.
    - Resolved context leaks in HTTP requests.
- **Windows Compatibility**: Fixed host resolution and path handling issues on Windows setups.

### Removed
- Obsolete `tools_format.go` and various dead code/residual files.

## [0.6.1] - 2026-02-20
- Initial beta release with multi-provider support.
- Basic tool calling and TUI components.
