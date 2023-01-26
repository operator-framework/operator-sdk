---
title: Opening Pull Requests
weight: 30
---

Changes to Operator SDK are submitted via [Github Pull Request (PR)][gh-pr]
to the relevant repository. Most PRs will be to the [main Operator SDK repo][operator-sdk-repo].

## PR Checklist
1. Add [tests][adding-tests] for your change, and ensure they run and pass in CI.
1. Add a [changelog entry][changelog-docs] if necessary.
1. Add relevant [documentation][adding-docs].
1. Rebase your commits on master and squash them into a single commit.
1. Write a concise commit message that references the issue number.
1. Push your commit to your fork and [open a pull request][gh-fork-pr].


## Review

Before a pull request can be merged, tests must pass in CI and it must be reviewed. A PR
must be approved by 2 reviewers, one of which must be at least at least a reviewer and one
of which must be at least an approver, per the [Operator Framework community guidelines][of-contributor-ladder].

Please feel free to message the developers to get eyes on your PR, whether through @'ing on the PR itself,
the #operator-sdk-dev channel on Kubernetes slack, or by attending the Operator SDK [triage meeting][triage-meeting].

[adding-docs]:https://sdk.operatorframework.io/docs/contribution-guidelines/documentation/
[adding-tests]:https://sdk.operatorframework.io/docs/contribution-guidelines/testing/
[changelog-docs]:https://sdk.operatorframework.io/docs/contribution-guidelines/changelog/#m-docscontribution-guidelineschangelog
[gh-fork-pr]:https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request-from-a-fork
[gh-pr]:https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/about-pull-requests
[of-contributor-ladder]:https://github.com/operator-framework/community/blob/master/contributor-ladder.md
[operator-sdk-repo]:https://github.com/operator-framework/operator-sdk
[triage-meeting]:https://github.com/operator-framework/community#operator-sdk-issue-triage
