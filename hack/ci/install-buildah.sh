# Install buildah and runtime dependencies.
sudo apt-get -y update
sudo apt-get -y install software-properties-common
sudo add-apt-repository -y ppa:alexlarsson/flatpak
sudo add-apt-repository -y ppa:gophers/archive
sudo apt-add-repository -y ppa:projectatomic/ppa
sudo apt-get -y -qq update
sudo apt-get -y install \
    btrfs-tools \
    libapparmor-dev \
    libdevmapper-dev \
    libglib2.0-dev \
    libgpgme11-dev \
    libostree-dev \
    seccomp \
    libseccomp-dev \
    libselinux1-dev \
    skopeo-containers \
    go-md2man

# Build and install buildah to /usr/local/bin
git clone https://github.com/containers/buildah ${GOPATH}/src/github.com/containers/buildah
cd ${GOPATH}/src/github.com/containers/buildah
make runc all TAGS="apparmor seccomp"
sudo make install install.runc

# Rootless builds require users to have entries in /etc/sub{u,g}id
sudo sh -c "echo \"$USER:100000:65536\" >> /etc/subuid"
sudo sh -c "echo \"$USER:100000:65536\" >> /etc/subgid"

# buildah expects search registries in /etc/containers/registries.conf
cat <<EOF > registries.conf
[registries.search]
registries = ['docker.io']
EOF
[ ! -e /etc/containers ] && sudo mkdir /etc/containers
sudo mv registries.conf /etc/containers/

# Confirm buildah was built correctly
buildah --version
