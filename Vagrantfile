# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure(2) do |config|
    config.vm.box = "debian/bookworm64"
    config.vm.hostname = "debian12"
    config.vm.provider "virtualbox" do |v|
        v.name = "debian12"
        v.memory = 1024
        v.cpus = 1
        v.customize ["modifyvm", :id, "--natdnsproxy1", "on"]
        v.customize ["modifyvm", :id, "--natdnshostresolver1", "on"]
        v.customize ["modifyvm", :id, "--uartmode1", "file", File::NULL]
    end
    config.vm.network "forwarded_port", guest: 8443, host: 8443
    config.vm.synced_folder ".", "/vagrant", SharedFoldersEnableSymlinksCreate: true
    config.vm.provision "shell", inline: <<-SHELL
# Setup Base Packages
ACCEPT_EULA=Y DEBIAN_FRONTEND=noninteractive apt-get update && \
    apt-get upgrade -y && \
    apt-get dist-upgrade -y && \
    apt-get install -y --no-install-recommends \
        curl \
        ca-certificates \
        tzdata \
        htop \
        jq && \
    apt-get clean

# Install Docker-CE Engine
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
rm -f ./get-docker.sh

# Setup pal
dpkg -i /vagrant/pal*amd64.deb

# Add pal to Docker group
usermod -aG docker pal

# Create Self-Signed Certs
openssl req -x509 -newkey rsa:4096 -nodes -keyout /etc/pal/localhost.key -out /etc/pal/localhost.pem -days 365 -sha256 -subj '/CN=localhost' -addext "subjectAltName=IP:127.0.0.1,DNS:localhost"

# Copy Insecure Test Configs
cp -f /vagrant/test/pal.yml /etc/pal/
cp -f /vagrant/test/*.yml /etc/pal/actions/

# Ensure permissions are correct
chown -Rf pal:pal /etc/pal /pal

# Run pal Systemd Service
systemctl daemon-reload
systemctl enable pal
systemctl restart pal
SHELL
end
