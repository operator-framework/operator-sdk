---
title: Backport Policy
weight: 30
---

Usually, just very critical issues are backport. Also, it is just done to the most recent release. This kind of decision and discussion can get in place during the Triage meeting. For further information about how contact the team and attending the meetings check the [community](https://github.com/operator-framework/community) repository.   

Note that, if maintainers run across backport-able issues while working, it is possible to immediately decide to backport it. And then, the process would be track the issue in the repository and to do two pull requests with the fix where one should be made against the master branch and the other should be a cherry-pick against the most recent release branch where it will be backported.
