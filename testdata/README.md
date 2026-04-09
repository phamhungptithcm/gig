# Test Data

Current tests create temporary repositories and workspaces at runtime instead of storing static Git fixtures in the repository.

This keeps the test suite portable and avoids committing repository metadata into source control.

CLI golden output fixtures live under package-local `testdata/` folders, such as `internal/cli/testdata/`.
