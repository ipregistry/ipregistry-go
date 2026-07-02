# Contributing

Thanks for your interest in improving the official Ipregistry Go client.

## Development

The library targets Go 1.23+ and has **no external dependencies** — please keep it that way (standard library only).

Common tasks are wired through the [`Makefile`](Makefile):

```bash
make test        # run the test suite
make race        # run tests with the race detector
make cover       # print total coverage
make vet         # go vet
make fmtcheck    # fail if any file is not gofmt-clean
make lint        # staticcheck (install: go install honnef.co/go/tools/cmd/staticcheck@latest)
make all         # fmtcheck + vet + test
```

Before opening a pull request, please make sure `make all` passes and code is `gofmt`-formatted.

## Guidelines

- **Public API stability.** This module is `v1`; avoid breaking changes to exported identifiers. If a breaking change is
  unavoidable, it must go through a new major version (a `/v2` module path).
- **Naming.** Follow Go conventions, including all-caps initialisms (`IP`, `API`, `ASN`, `URL`, `ID`).
- **Errors.** Surface API failures as `*APIError` and client-side failures as `*ClientError`; wrap underlying causes so
  `errors.Is`/`errors.As` keep working.
- **Tests.** Add or update tests for any behavior change. Offline behavior is tested with `net/http/httptest`; no live
  API key is required. Live system tests live behind the `integration` build tag and run against the real API when
  `IPREGISTRY_API_KEY` is set (`make integration`); they consume credits, so keep them minimal.
- **Docs.** Keep the `README.md`, doc comments, and runnable examples in `examples/` in sync with the code.
- **Changelog.** Record notable changes in `CHANGELOG.md` under `[Unreleased]`.

## Reporting issues

For bugs or feature requests, please open a GitHub issue. For account or API questions, contact
[support@ipregistry.co](mailto:support@ipregistry.co).
