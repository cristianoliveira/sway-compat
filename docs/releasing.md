# Release process

Releases are Linux binaries published by GitHub Actions. Each release contains `amd64` and `arm64` tarballs plus SHA-256 checksums.

## Prepare

1. Ensure `main` is clean and current:

   ```bash
   git switch main
   git pull --ff-only
   git fetch origin main
   git merge-base --is-ancestor HEAD origin/main
   go test ./...
   ```

2. Choose a semantic version. Use a pre-release suffix while the project is unstable, for example `v0.2.0-rc.1`.
3. Review commits since the previous tag:

   ```bash
   git log --oneline "$(git describe --tags --abbrev=0 2>/dev/null || git rev-list --max-parents=0 HEAD)..HEAD"
   ```

4. Optionally build release-equivalent artifacts locally:

   ```bash
   make release-snapshot
   rm -rf /tmp/sway-compat-snapshot
   mkdir -p /tmp/sway-compat-snapshot
   tar -xzf "$(find dist -maxdepth 1 -name '*linux_amd64.tar.gz' -print -quit)" -C /tmp/sway-compat-snapshot
   /tmp/sway-compat-snapshot/sway-compat --version
   ```

## Publish

Create and push an annotated tag from `main`:

```bash
VERSION=v0.2.0
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"
```

The `Release` workflow tests the tag, builds static Linux binaries, creates checksums, generates release notes from commits, and publishes the GitHub release. A version containing a suffix such as `-rc.1` becomes a pre-release.

Verify the workflow and artifacts:

```bash
gh run list --workflow Release --limit 1
gh release view "$VERSION"
gh release download "$VERSION" --pattern checksums.txt --pattern '*.tar.gz' --dir /tmp/sway-compat-release
(cd /tmp/sway-compat-release && sha256sum --check checksums.txt)
```

Extract the archive for your architecture and confirm the embedded version:

```bash
tar -xzf "/tmp/sway-compat-release/sway-compat_${VERSION#v}_linux_amd64.tar.gz" -C /tmp/sway-compat-release
/tmp/sway-compat-release/sway-compat --version
```

## Recover from a bad release

Do not move or reuse a published tag. Delete the bad release and tag, fix the issue, then publish a new patch version:

```bash
gh release delete "$VERSION" --yes
git push origin --delete "$VERSION"
git tag --delete "$VERSION"
```
