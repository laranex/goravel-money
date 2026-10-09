# Contribution Guide

Thank you for considering contributing to Goravel Money! Please review the following guidelines before submitting a pull request.

For significant changes, please open an issue first so we can discuss the approach.

## Process

1. Fork the project
2. Create a new branch
3. Code, test, commit, and push
4. Open a pull request detailing your changes

## Guidelines

- Ensure the code is formatted with `gofmt` and passes `go vet ./...`.
- Amounts are exact integer minor units of any size (`math/big`); never introduce a `float64` into an amount path.
- Currency data and minor units follow ISO 4217; parsing and formatting stay in line with `laranex/laravel-money` (moneyphp), so add the same test vector to both packages when behavior changes.
- Send a coherent commit history, making sure each commit in your pull request is meaningful.
- You may need to [rebase](https://git-scm.com/book/en/v2/Git-Branching-Rebasing) to avoid merge conflicts.
- Please remember that we follow [SemVer](http://semver.org/); the import path carries the major version (`/v4`).

## Setup

Clone your fork; Go 1.25 or higher is the only requirement:

```bash
go version
```

## Lint

Format and vet your code:

```bash
gofmt -l .
go vet ./...
```

## Tests

Run all tests:

```bash
go test -race ./...
```
