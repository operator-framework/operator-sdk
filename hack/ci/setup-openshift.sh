# Configure insecure docker registry for openshift
sudo service docker stop
sudo sed -i 's/DOCKER_OPTS=\"/DOCKER_OPTS=\"--insecure-registry 172.30.0.0\/16 /' /etc/default/docker
sudo service docker start
# Download oc to spin up openshift on local docker instance
curl -Lo oc.tar.gz https://github.com/openshift/origin/releases/download/v3.11.0/openshift-origin-client-tools-v3.11.0-0cbc58b-linux-64bit.tar.gz
# Put oc binary in path
tar xvzOf oc.tar.gz openshift-origin-client-tools-v3.11.0-0cbc58b-linux-64bit/oc > oc && chmod +x oc && sudo mv oc /usr/local/bin/
# Start oc cluster
oc cluster up
# Become cluster admin
oc login -u system:admin

# kubectl is needed for the single namespace local test and the ansible tests
curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.11.3/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
