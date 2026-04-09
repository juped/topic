# topic

Git's commit graph *is* a topic dependency graph; `topic` keeps it consistent.

Most teams use feature branches and pull requests in an ad-hoc way which 
obfuscates useful history and makes dependency relationships implicit
at best. This isn't particularly a failing of these teams; it's what the 
GitHub UI pushes them towards.

`topic` is a workflow tool based on `gitworkflows(7)` and the mailing-list 
patch series model, but adapted for engineering teams who want the
benefits of that workflow without having to actually use emailed patches or 
have a full-time repository maintainer.

## Installation

```sh
go install go.topic.tools/topic@latest
```

Requires Git 2.38+.

## Concepts

A **topic** is a named branch representing a unit of work. Topics can depend
on other topics, and that dependency is structural — encoded in the commit
graph — not just a comment in a PR description.

```sh
# Start a new topic off the latest release tag
topic create ray/my-feature

# Start a topic that builds on another
topic create ray/dependent-feature -d alice/other-feature
```

The dependency relationship between topics is readable directly from the git
object graph; this part of the tool requires no additional metadata.

## The problem with the GitHub workflow

The standard GitHub workflow — branch off `main`, open a PR, merge when green
— has a subtle structural flaw that compounds as teams grow.

When a branch is cut from an arbitrary point on `main`, it implicitly includes
everything that was on `main` at that moment: other people's features,
half-landed refactors, whatever happened to be there. The branch isn't really
"my feature" — it's "my feature plus a random snapshot of everyone else's
work." When it comes time to integrate, you're not merging a topic. You're
merging two large, ill-defined sets of changes, and conflicts between them
could originate anywhere in either set.

The key conceptual error is thinking that a branch "has conflicts." It doesn't.
Two branches can be *pairwise* in conflict — this change and that change don't
compose — but in the GitHub workflow that pairwise relationship is buried under
layers of coincidental inclusion. Whoever merges second inherits the conflict,
with no clear indication of which of the dozens of commits in each branch
actually caused it, and no systematic way to record the resolution so it
doesn't have to be relitigated next time.

`topic` enforces the discipline that prevents this: a topic branch contains
exactly its topic, with explicit dependencies on other topics it requires.
The commit graph becomes a real dependency graph, not an accident of timing.
Pairwise conflicts between topics are detectable early, attributable clearly,
and resolvable once.

## The problem with `rerere`

Git's `rerere` (reuse recorded resolution) is underused but genuinely useful:
it records how you resolved a merge conflict so it can replay that resolution
automatically next time the same conflict appears.

The problem is that `rerere` only activates on *textual* conflicts — hunks
that git's merge machinery can't combine automatically. It misses the more
dangerous class of conflicts: two changes that merge cleanly but break things
together. A function rename in one branch, a new call site in another. Both
apply without conflict markers. Neither works without the other.

`topic` implements **semantic rerere**: it records pre/post images for *all*
topic integrations, not just the ones that produce conflict markers. 
Semantic rerere does require additional metadata; unlike git's native 
`rerere`, the cache is stored in a `_topic-metadata` branch and distributed 
alongside your normal git push/fetch workflow, so the whole team shares 
reconciliation knowledge.

## Status

`topic create` is working and usable today. Semantic rerere (`topic reconcile`)
and the release candidate builder are under active development.

## Roadmap

- `topic reconcile` - record and replay semantic reconciliations
- `topic integrate` - build a conflict-free release candidate
