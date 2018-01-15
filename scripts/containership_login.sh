#!/bin/sh

USER=$1
USER_HOME=/etc/containership/home/$USER

if [ $# -eq 0 ]; then
    exit 1
fi

if ! id $USER > /dev/null 2>&1; then
    if command -v useradd > /dev/null 2>&1; then
        sudo useradd -m -d $USER_HOME $USER
    elif command -v adduser > /dev/null 2>&1; then
        sudo adduser -h $USER_HOME -D $USER > /dev/null 2>&1
        sudo passwd -u $USER > /dev/null 2>&1
    else
        exit 1
    fi

    echo "$USER ALL=(ALL) NOPASSWD:ALL" | sudo EDITOR="tee -a" visudo > /dev/null 2>&1
fi

sudo su - $USER
