# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure(2) do |config|
    config.vm.box = "debian/bookworm64"
    config.vm.hostname = "debian12"
    config.vm.provider "virtualbox" do |v|
        v.name = "debian12"
        v.memory = 2048
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
        jq && \
    apt-get clean

# Setup pal
dpkg -i /vagrant/pal*amd64.deb

# Create Self-Signed Certs
cd /pal
openssl req -x509 -newkey rsa:4096 -nodes -keyout /pal/localhost.key -out /pal/localhost.pem -days 365 -sha256 -subj '/CN=localhost' -addext "subjectAltName=IP:127.0.0.1,DNS:localhost"
chown -Rf pal:pal /pal

# Copy Insecure Test Configs
cp -f /vagrant/pal.yml /pal/
cp -f /vagrant/test/*.yml /pal/actions/

# Configure Paths for /pal In pal.yml
sed -i "s|listen:.*|listen: 0.0.0.0:8443|" /pal/pal.yml
sed -i "s|  key:.*|  key: /pal/localhost.key|" /pal/pal.yml
sed -i "s|cert:.*|cert: /pal/localhost.pem|" /pal/pal.yml
sed -i "s|upload_dir:.*|upload_dir: /pal/upload|" /pal/pal.yml
sed -i "s|path:.*|path: /pal/pal.db|" /pal/pal.yml

# Run pal Systemd Service
systemctl daemon-reload
systemctl enable pal
systemctl restart pal
SHELL
end
