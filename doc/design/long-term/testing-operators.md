## Make testing an Operator simple and easy

**Goal:** Make local development of Operators easy, so software vendors can produce high quality software. 
Make functional tests easy to integrate into a big matrix (upstream, OpenShift, GKE, etc) since we want the SDK to work equally well on all platforms.

**Problem:** Operators are inherently tied to a clustered environment, which complicates testing. Running a full cluster locally can be hard, and requiring a remote cluster is clunky. Aside from a single test cluster, running your Operator against the range of Kubernetes offering requires a ton of industry knowledge and expertise.
 
**Why is this important:** We think Operators are the best way to ship software on Kubernetes. This will naturally happen if we provide methods to build high quality software, of which testing is a huge part.

**Proposed work streams to accomplish this:**
 - Mock clients to enable unit testing without Kubernetes in under a second
 - Functional testing is run against X last releases via platform scripts in codebase
 - SDK provides a method to generate a functional testing container
 - Docs for using a test container and setting it up against a number of providers for use in a partners CI environment
 - Provide a way to run the Operator binary locally outside of a container
    - Eg. https://github.com/operator-framework/operator-sdk/issues/142

**Open questions:**
  - none so far
