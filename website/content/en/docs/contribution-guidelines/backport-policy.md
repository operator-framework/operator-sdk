---
title: Backport Policy
weight: 30
---

Usually, only very critical issues are backported. Also, it is just done to the most recent release. This can be discussed and decided during the weekly Triage meeting. For further information about how to contact the team and attending the meetings, check the [community](https://github.com/operator-framework/community) repository.   

Note that, if maintainers run across backport-able issues while working, it is possible to immediately decide to backport it. And then, the process would be to track the issue in the repository, and to do two pull requests with the fix. One should be made against the master branch, and the other should be a cherry-pick against the most recent release branch, where it will be backported.
