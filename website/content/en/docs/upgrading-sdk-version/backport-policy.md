---
title: Backport Policy
weight: 2
---

Mainly critical issue fixes are backported to the most recent minor release.
Special backport requests can be discussed during the weekly Triage meeting; this does not guarantee an exceptional backport will be created.
Occasionally non-critical issue fixes will be backported, either at an approver's discretion or by request as noted above.
For information on contacting maintainers and attending meetings, check the [community](https://github.com/operator-framework/community) repository.   

## Process

Typically an issue will be fixed in the `master` branch, which will then be cherry-picked to the most recent release's branch.
Those with approver permissions and above can create a cherry-pick PR, assuming no conflicts, by commenting `/cherry-pick <release branch>`
in the PR fixing the issue in master. Fixes that are only relevant to a specific release branch can be made against
that branch directly.
