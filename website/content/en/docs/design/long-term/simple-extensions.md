## Simple Operator Extensions

**Goal:** ISVs choose the Operator SDK to ship their apps because it provides OOTB extensions for common enterprise asks, like audit logging, handling TLS, monitoring, etc. The SDK provides these capabilities as libraries that can be easily inserted into your business logic where it makes sense.

**Problem:** Kubernetes provides the unified stack that software vendors strive for. However, each vendor is left to writing code to utilize platform features. This leads to custom solutions, incorrect usage and security flaws. It’s impossible for one engineering team to be an expert in logging, monitoring, security, etc in addition to software they are providing.

**Why is this important:** The SDK should be solving hard problems and saving folks time. Vendors have different needs, and should be able to plug and play different features as they require them.

**Proposed work streams to accomplish this:**
1. Provide methods for generating and storing TLS assets as secrets
2. Operator emits Prometheus metrics by default and optional other ones can be created
3. Ship default logging for generated code and methods for adding more
4. Authentication/Authorization
	- example: deploy a database and provide access to a specific set of Kubernetes users with the same credentials
5. Backup/Restore hooks
	- example: before/after upgrade
6. Provide hooks into the Kubernetes audit log and event stream

**Open questions:**
  - none so far