---
title: Upgrade SDK Version 
weight: 4
description: Guide to upgrading sdk version for your operator
---

## Backwards Compatibility when Upgrading Operator-sdk version

When upgrading your version of Operator-sdk, it is intended that post-1.0.0 minor versions (i.e. 1.y) are backwards compatible and strictly additive. Therefore, you
only need to re-scaffold your operator with a newer version of Operator-SDK if you wish to take advantage of new features. If you do not wish to use new features,
all that should be required is bumping the operator image dependency (if a Helm or Ansible operator) and rebuilding your operator image.
