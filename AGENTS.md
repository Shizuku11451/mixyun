# AGENTS.md

Mixyun is a fork of MixMusic (`upstream`: fzn0x/mixmusic). Its own
repository is `origin`: Shizuku11451/mixyun.

## Landing changes

Every change reaches `main` through a pull request, never by pushing `main`:

1. Commit on a branch, never on `main`.
2. Push the branch to `origin`: `git push -u origin <branch>`.
3. Open the pull request against the fork's `main`, naming the repository:
   `gh pr create --repo Shizuku11451/mixyun --base main --head <branch>`.
   Without `--repo`, `gh` opens it against the upstream MixMusic.
4. Run `go test ./...` before you push, then merge once CI's three Go jobs
   (`Go (ubuntu-latest)`, `Go (windows-latest)`, `Go (macos-latest)`) pass,
   about 10 minutes. Build and Flatpak, which come after, only package the
   app: wait for them only when the change touches the packaging
   (`packaging/`, `.github/`, `flake.nix`, the build scripts). `main` has no
   branch protection: a red CI does not stop the merge, so check it first.
   Leave `ci.yml` as upstream has it: it changes there often, and an edit of
   the fork's would conflict at each sync.

Pull requests to upstream MixMusic are separate work, on their own branches,
with `--repo fzn0x/mixmusic`.

## MyGo's gaps

MixMusic draws with [MyGo](https://github.com/egoist/mygo). When MyGo lacks
something MixMusic needs, has a bug, or does work it could skip, add an entry
to [MYGO.md](MYGO.md) as you find it, even when you work around it: what is
missing, where in MyGo (file and function, with the version from `go.mod`),
how it shows in MixMusic, and the workaround, if any. These entries become
PRs to MyGo later. Read the MyGo source in the module cache
(`go env GOMODCACHE`) before you write one: only what you checked goes in.
