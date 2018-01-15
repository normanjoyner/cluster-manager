#!/bin/sh

USER=$1
USER_HOME=/etc/containership/home/$USER

# Bail if no args
if [[ $# -eq 0 ]]; then
    echo "ERROR: User argument required"
    exit 1
fi

# Create user if it doesn't exist
if ! id $USER > /dev/null 2>&1; then
    if command -v useradd > /dev/null 2>&1; then
        echo "Creating user via useradd: $USER"
        sudo useradd -d $USER_HOME -s /bin/bash $USER
    elif command -v adduser > /dev/null 2>&1; then
        sudo adduser -h $USER_HOME -D $USER > /dev/null 2>&1
        sudo passwd -u $USER > /dev/null 2>&1
    else
        echo "No way to create user. Exiting."
        exit 1
    fi

    echo "$USER ALL=(ALL) NOPASSWD:ALL" | sudo EDITOR="tee -a" visudo > /dev/null 2>&1
fi

echo "Dropping into user shell"
sudo su - $USER
