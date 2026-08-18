# Changelog

## [v0.5.0](https://gitlab.com/phpboyscout/go/errorhandling/-/releases/v0.5.0)

[Compare to previous version](https://gitlab.com/phpboyscout/go/errorhandling/-/compare/v0.4.0...v0.5.0)

### Features

- **errors**: stop masking an error's kind, and share the unknown-verb wording ([d859411](https://gitlab.com/phpboyscout/go/errorhandling/-/commit/d859411b7d787369e5c957a6a36ed4e86fc6e097))

### Bug Fixes

- **deps**: update module github.com/stretchr/testify to v1.12.0 ([ea4abc8](https://gitlab.com/phpboyscout/go/errorhandling/-/commit/ea4abc8ff05a16c212f072737f2954905af5b5f5))

## [v0.4.0](https://gitlab.com/phpboyscout/go/errorhandling/-/releases/v0.4.0)

[Compare to previous version](https://gitlab.com/phpboyscout/go/errorhandling/-/compare/v0.3.0...v0.4.0)

### Features

- **errors**: add ErrUnknownSubCommand for a mistyped subcommand ([832cca2](https://gitlab.com/phpboyscout/go/errorhandling/-/commit/832cca2ba0d5d168cdebf620b6c0ac68af29589e))

### Bug Fixes

- **reporting**: keep the error structured under our own wrappers ([9168455](https://gitlab.com/phpboyscout/go/errorhandling/-/commit/9168455a743bf22a171a5a50af5ffdeeb982b71d))
- **mocks**: regenerate MockErrorHandler against the current interface ([8299dd9](https://gitlab.com/phpboyscout/go/errorhandling/-/commit/8299dd9c4f3af8184ce0271ec32fec798ffe11f4))
- **deps**: require go 1.26.6 for the stdlib advisories ([288f740](https://gitlab.com/phpboyscout/go/errorhandling/-/commit/288f74081c9d154e19eee5726a38078321263118))
- **ci**: bump the cicd components to v0.36.0 for Go 1.26.6 ([8fc5ef6](https://gitlab.com/phpboyscout/go/errorhandling/-/commit/8fc5ef6cfb1d35ef61aec5953ab15779e15150c5))

## [v0.3.0](https://gitlab.com/phpboyscout/go/errorhandling/-/releases/v0.3.0)

[Compare to previous version](https://gitlab.com/phpboyscout/go/errorhandling/-/compare/v0.2.0...v0.3.0)

### Features

- **stack**: report the stack where the error began ([c4f2ef4](https://gitlab.com/phpboyscout/go/errorhandling/-/commit/c4f2ef435936399acbdc085474efbfc3fe435826))

## [v0.2.0](https://gitlab.com/phpboyscout/go/errorhandling/-/releases/v0.2.0)

[Compare to previous version](https://gitlab.com/phpboyscout/go/errorhandling/-/compare/v0.1.1...v0.2.0)

### Features

- the handler after cockroachdb ([7c46963](https://gitlab.com/phpboyscout/go/errorhandling/-/commit/7c46963443b3e3d3c05bc3c7a29245fe7401b06e))

## [v0.1.1](https://gitlab.com/phpboyscout/go/errorhandling/-/releases/v0.1.1)

[Compare to previous version](https://gitlab.com/phpboyscout/go/errorhandling/-/compare/v0.1.0...v0.1.1)

### Bug Fixes

- **errorhandling**: exit non-zero on fatal special errors; drop dead WithWriter ([77cc66b](https://gitlab.com/phpboyscout/go/errorhandling/-/commit/77cc66bfd065a8ce8abca83fdec2b6ff0985a728))

## [v0.1.0](https://gitlab.com/phpboyscout/go/errorhandling/-/releases/v0.1.0)

### Features

- extract the error-reporting layer from go-tool-base
