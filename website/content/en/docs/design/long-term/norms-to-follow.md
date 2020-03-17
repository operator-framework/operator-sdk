## SDK examples/docs show API norms authors should follow

**Goal:** Help Operator authors avoid common mistakes and usability issues that the community has learned through the various operators in use today.

**Problem:** There are three classes of mistakes that we have seen folks make:
1. Methods for connecting Kubernetes objects through labels, OwnerRefs, status etc that may be incompatible with future updates or too restrictive in the long term.
2. Attempting to add in remote connectivity directly into the Operator via an HTTP API. The Kubernetes API should be the source of desired state.
3. Requiring state that is not stored within the cluster to operate safely.

**Why is this important:** The SDK will be successful if it solves problems for developers out of the box. Shipping updates and new features safely via an Operator is only possible when they are deployed and used correctly.

**Proposed work streams to accomplish this:**
 - Documentation for dos and don’ts for an Operator
 - Canonical sample operator that shows a complex use-case done correctly

**Open questions:**
1. How much of the object organization is done via helper code in the SDK vs well-planned label queries and OwnerRefs by the developer?