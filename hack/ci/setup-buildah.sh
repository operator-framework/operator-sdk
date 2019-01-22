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
