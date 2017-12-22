# -*- mode: ruby -*-
# vi: set ft=ruby :

SRC_DIR_HOST = "."
# TODO make this use $HOME on VM or maybe don't use $HOME at all
SRC_DIR_VM = "/home/containership/go/src/github.com/containership/cloud-agent"

# Hack to sync a directory owned by a user that doesn't exist on VM yet
# https://stackoverflow.com/a/19682290
CS_UID="1001"
CS_GID="1001"

GO_VERSION="1.9.2"

Vagrant.configure("2") do |config|
  # TODO more base images
  config.vm.box = "ubuntu/xenial64"
  GO_PLATFORM="linux"
  GO_ARCH="amd64"

  # ssh as containership only when `vagrant ssh` is used (to not interfere
  # with provisioning)
  VAGRANT_CMD = ARGV[0]
  if VAGRANT_CMD == "ssh"
    config.ssh.username = 'containership'
    config.ssh.private_key_path = File.join(Dir.home, ".ssh", "id_rsa")
  end

  # Sync the repo
  config.vm.synced_folder SRC_DIR_HOST, SRC_DIR_VM,
    owner: CS_UID, group: CS_GID

  # Inject our own public key
  # Assumes that the key pair we're interested in lives at the canonical
  # location on host
  public_key_path = File.join(Dir.home, ".ssh", "id_rsa.pub")
  public_key = IO.read(public_key_path)

  config.ssh.forward_agent = true
  config.ssh.keys_only = true

  # Run bootstrap script
  # TODO ideally we'd just call the prod bootstrap stuff here when it exists
  config.vm.provision "shell" do |s|
    s.path = "./scripts/vagrant/bootstrap.sh"
    s.args = [SRC_DIR_VM, public_key, CS_UID, CS_GID, GO_VERSION, GO_PLATFORM, GO_ARCH]
  end
end
