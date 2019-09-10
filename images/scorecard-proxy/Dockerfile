# Base image Proxy
# ----------------
#############################################
# Temp image to build the prerequisites 
#############################################
FROM registry.access.redhat.com/ubi8/ubi:latest AS builder

RUN groupadd -g 1001 scorecard-proxy && \
    useradd -m -s /sbin/nologin -u 1001 -g scorecard-proxy scorecard-proxy

#############################################
# Container Image
#############################################
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
MAINTAINER Red Hat Operator Framework <operator-framework@googlegroups.com>

# Define Labels
LABEL name="operator-framework/operator-sdk-proxy-image" \
      maintainer="operator-framework@googlegroups.com" \
      vendor="Operator Framework" 

# Define EnvVars
ENV PROXY=/usr/local/bin/scorecard-proxy \
    USER_UID=1001 \
    USER_NAME=scorecard-proxy \
    USER_HOME=/home/scorecard-proxy  

# Copy group and user created in the builder image
COPY --from=builder /etc/passwd /tmp
COPY --from=builder /etc/group /tmp

# Create the user and group in this image and add permissions 
RUN cat /tmp/passwd | grep ${USER_NAME} >> /etc/passwd \
    && cat /tmp/group | grep ${USER_NAME} >> /etc/group \
    && mkdir -p ${USER_HOME} \
    && chown -R ${USER_NAME}:${USER_NAME} ${USER_HOME} \
    && chmod ug+rwx ${USER_HOME} \
    && rm -rf /tmp/passwd \
    && rm -rf /tmp/group

# Use the user created to run the container as rootless
USER ${USER_UID}

# Installs the operator binary
COPY build/_output/bin/${USER_NAME} ${PROXY}
COPY build/bin /usr/local/bin

# This allows the Platform to validate the authority the image
# More info: https://docs.openshift.com/container-platform/3.11/creating_images/guidelines.html#openshift-specific-guidelines
ENTRYPOINT ["/usr/local/bin/entrypoint"]

# Execute the operator 
CMD exec ${PROXY} $@
