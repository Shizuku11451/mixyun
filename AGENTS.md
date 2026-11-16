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
4. Wait for CI (`ci.yml`, Go on Linux, Windows and macOS) to pass, then merge.
   `main` has no branch protection: a red CI does not stop the merge, so check
   it first.

Pull requests to upstream MixMusic are separate work, on their own branches,
with `--repo fzn0x/mixmusic`.
