## Manage Dockerfiles

### Goal
The Operator-SDK should allow users to create multiple images with different base images easily. We want the users to be able to decide dynamically choose the base image. The Operator-SDK should be able to change and own the steps that it needs to take for whichever type of operator. 

### Non-Goals
* Allow users to change the flow of how images are created
* Support users to add actions/steps to the end of the image creation.

### Proposed Solution

#### User-facing changes
The `operator-sdk build` command will add an option to pass in the base image example: `operator-sdk build --base-dockerfile centos-base.Dockerfile`. The `new` command will generate a default base, as it does now, called `base.Dockerfile`. If no option is passed in `base.Dockerfile`  will be selected.

#### Operator-SDK Changes
The `operator-sdk` will now add an `sdk.Dockerfile`. This dockerfile will contain all the steps to set up the operator image correctly for whichever type of operator this is. This will be changed/updated and owned by the operator-sdk, and we will not expect or respect any changes made by a user to this file. This will allow us to completely re-write it as we see fit. The testing framework will not be changed as part of this proposal.

The SDK will layer the images together. First, it will build an image from the `base` dockerfile. The sdk owned dockerfiles will use `FROM-ARG` to use the base image as its base and then create an image from this. This will behave similarly to how the building with test works now. The test dockerfile will also be added if needed at the appropriate time. 


#### Hybrid Approach
Because the operator-sdk owns `sdk.Dockerfile`, when moving to a hybrid approach, we can update the `sdk.Dockerfile` to use the combination of the dockerfiles.


