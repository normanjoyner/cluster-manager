#!/bin/bash

USER=$1
USER_HOME=/etc/containership/home/$USER

if [ ! -d $USER_HOME ]; then
    echo "Creating user home at $USER_HOME"
    sudo mkdir $USER_HOME
fi

if ! getent passwd $USER > /dev/null 2>&1; then
    echo "Creating user $USER"
    sudo useradd -d $USER_HOME -s /bin/bash $USER

    echo "Fixing ownership of $USER_HOME"
    sudo chown $USER:$USER $USER_HOME
fi

echo "Dropping into user shell"
sudo su - $USER
