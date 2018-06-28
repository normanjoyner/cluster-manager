#!/bin/bash

# TODO hopefully most of this script dies one day
# in favor of prod provisioning script
# TODO this should all be idempotent in case `vagrant up` runs twice

# TODO better argument parsing or use env vars
SRC_DIR=$1
PUBLIC_KEY=$2
CS_UID=$3
CS_GID=$4
GO_VERSION=$5
GO_PLATFORM=$6
GO_ARCH=$7
echo SRC_DIR=$SRC_DIR
echo PUBLIC_KEY=$PUBLIC_KEY

createBashProfile () {
    echo "Creating .bash_profile..."
    cat << EOF > /home/containership/.bash_profile
export PATH=\$PATH:/usr/local/go/bin
export GOPATH=\$HOME/go
export SRC_DIR=$SRC_DIR

cd $SRC_DIR
EOF
}

# Prevent errors from debconf
export DEBIAN_FRONTEND=noninteractive

# Install golang
GO_ARCHIVE=go$GO_VERSION.$GO_PLATFORM-$GO_ARCH.tar.gz
GO_DL_URL=https://storage.googleapis.com/golang/$GO_ARCHIVE
GO_INSTALL_DIR=/usr/local
echo "Downloading golang from $GO_DL_URL..."
curl -s -O $GO_DL_URL
echo "Installing golang to $GO_INSTALL_DIR..."
tar -C /usr/local -xf $GO_ARCHIVE

echo "Installing additional tools..."
# TODO pin versions
add-apt-repository ppa:masterminds/glide -y
apt-get update -y
apt-get install make docker.io glide -y > /dev/null 2>&1

if ! getent group containership > /dev/null 2>&1; then
    echo "Creating containership group..."
    groupadd --gid $CS_GID containership
else
    echo "Group containership already exists"
fi

if ! getent passwd containership > /dev/null 2>&1; then
    echo "Creating containership user..."
    useradd --uid $CS_UID --shell /bin/bash -g containership --groups sudo,docker containership
    CS_HOME=$(getent passwd containership | cut -d: -f6)

    echo "Creating authorized_keys..."
    mkdir -p $CS_HOME/.ssh
    AUTHORIZED_KEYS=$CS_HOME/.ssh/authorized_keys
    echo "$PUBLIC_KEY" > $AUTHORIZED_KEYS

    chmod 600 $AUTHORIZED_KEYS

    # give containership user passwordless sudo
    echo 'containership   ALL = NOPASSWD: ALL' >> /etc/sudoers

    createBashProfile

    chown -R containership:containership $CS_HOME
else
    echo "User containership already exists"
fi
