#!/bin/bash

USER=$1
USER_HOME=/etc/containership/home/$USER

# Bail if no args
if [[ $# -eq 0 ]]; then
    echo "ERROR: User argument required"
    exit 1
fi

# Create user if it doesn't exist
if ! getent passwd $USER > /dev/null 2>&1; then
    echo "Creating user $USER"
    sudo useradd -d $USER_HOME -s /bin/bash $USER

    echo "Adding passwordless sudo for $USER"
    sudo sh -c "echo '$USER ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/$USER"
fi

echo "Dropping into user shell"
sudo su - $USER
