# CentOS 9 Deployment Guide for Tiris Backend

This guide covers the specific considerations and steps for deploying Tiris Backend on CentOS 9 (Stream), Rocky Linux 9, or AlmaLinux 9.

## 🎯 Quick Start for CentOS 9

The deployment scripts automatically detect your OS and handle CentOS-specific configurations. The process is identical to Ubuntu:

```bash
# VPS setup (run once)
curl -fsSL https://raw.githubusercontent.com/your-repo/tiris-backend/master/deployment/scripts/vps-setup.sh | sudo bash

# Quick deployment
git clone https://github.com/your-username/tiris-backend.git
cd tiris-backend
./scripts/quick-deploy.sh
```

## 🔧 CentOS-Specific Differences

### Package Management
| Ubuntu/Debian | CentOS/RHEL |
|---------------|-------------|
| `apt update` | `dnf update` |
| `apt install` | `dnf install` |
| `apt-get` | `dnf` or `yum` (legacy) |

### Firewall
| Ubuntu/Debian | CentOS/RHEL |
|---------------|-------------|
| `ufw` (Uncomplicated Firewall) | `firewalld` |
| `ufw allow 80/tcp` | `firewall-cmd --add-port=80/tcp` |

### Services
| Ubuntu/Debian | CentOS/RHEL |
|---------------|-------------|
| `service` or `systemctl` | `systemctl` |
| Same systemd commands | Same systemd commands |

### Security Updates
| Ubuntu/Debian | CentOS/RHEL |
|---------------|-------------|
| `unattended-upgrades` | `dnf-automatic` |

## 📦 CentOS 9 Prerequisites

### Enable EPEL Repository
The deployment script automatically enables EPEL, but you can do it manually:

```bash
sudo dnf install -y epel-release
```

### Required Packages (auto-installed)
```bash
sudo dnf install -y \
    curl git firewalld htop nano vim wget unzip \
    ca-certificates gnupg certbot python3-certbot-nginx \
    fail2ban logrotate policycoreutils-python-utils tar which
```

## 🔥 Firewall Configuration (CentOS)

The script automatically configures `firewalld`, but here are manual commands:

```bash
# Start and enable firewalld
sudo systemctl enable --now firewalld

# Configure basic rules
sudo firewall-cmd --permanent --zone=public --add-service=ssh
sudo firewall-cmd --permanent --zone=public --add-service=http
sudo firewall-cmd --permanent --zone=public --add-service=https
sudo firewall-cmd --permanent --zone=public --add-port=8080/tcp
sudo firewall-cmd --reload

# Check active rules
sudo firewall-cmd --list-all
```

### Firewall Management Commands
```bash
# Check firewall status
sudo firewall-cmd --state

# List all open ports
sudo firewall-cmd --list-ports

# Add a new port
sudo firewall-cmd --permanent --add-port=3000/tcp
sudo firewall-cmd --reload

# Remove a port
sudo firewall-cmd --permanent --remove-port=3000/tcp
sudo firewall-cmd --reload

# Check if a port is open
sudo firewall-cmd --query-port=8080/tcp
```

## 🔒 SELinux Considerations

CentOS 9 has SELinux enabled by default. The deployment handles basic SELinux considerations, but here are useful commands:

```bash
# Check SELinux status
sestatus

# Check for SELinux denials
sudo ausearch -m AVC -ts recent

# Allow Docker to work with SELinux
sudo setsebool -P container_manage_cgroup on

# If you need to temporarily disable SELinux (not recommended)
sudo setenforce 0

# Re-enable SELinux
sudo setenforce 1
```

### Common SELinux Issues and Solutions

**Docker containers can't access mounted volumes:**
```bash
# Fix volume permissions
sudo chcon -Rt svirt_sandbox_file_t /opt/tiris/data
sudo chcon -Rt svirt_sandbox_file_t /opt/tiris/logs
```

**Web server can't connect to backend:**
```bash
# Allow HTTP network connections
sudo setsebool -P httpd_can_network_connect 1
```

## 🚀 Docker Installation (CentOS)

The script uses Docker's official installation method, but here's the manual process:

```bash
# Remove old Docker versions
sudo dnf remove -y docker docker-client docker-client-latest docker-common docker-latest docker-latest-logrotate docker-logrotate docker-engine

# Install Docker repository
sudo dnf install -y yum-utils
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo

# Install Docker
sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Start and enable Docker
sudo systemctl enable --now docker

# Add user to docker group
sudo usermod -aG docker $USER
```

## 🔧 CentOS-Specific Troubleshooting

### Docker Service Issues
```bash
# Check Docker status
sudo systemctl status docker

# Restart Docker daemon
sudo systemctl restart docker

# Check Docker logs
sudo journalctl -u docker.service
```

### DNS Resolution Issues
```bash
# Check DNS configuration
cat /etc/resolv.conf

# Test DNS resolution
nslookup google.com

# If using NetworkManager
sudo systemctl restart NetworkManager
```

### Container Permission Issues
```bash
# Fix SELinux contexts for container volumes
sudo restorecon -Rv /opt/tiris/

# Set proper ownership
sudo chown -R 1001:1001 /opt/tiris/data
sudo chown -R 1001:1001 /opt/tiris/logs
```

### Firewall Issues
```bash
# Check if Docker created iptables rules
sudo iptables -L DOCKER

# Restart firewalld if needed
sudo systemctl restart firewalld

# Check firewall logs
sudo journalctl -u firewalld
```

### SSL Certificate Issues
```bash
# CentOS uses different paths sometimes
sudo ln -sf /etc/pki/tls/certs/ca-bundle.crt /etc/ssl/certs/ca-certificates.crt

# Update CA certificates
sudo update-ca-certificates
```

## 📊 System Monitoring (CentOS)

### System Resources
```bash
# Check system info
hostnamectl
cat /etc/os-release

# Memory usage
free -h
cat /proc/meminfo

# Disk usage
df -h
lsblk

# Network interfaces
ip addr show
nmcli device status
```

### Log Locations (CentOS)
```bash
# System logs
sudo journalctl -f

# Docker logs
sudo journalctl -u docker -f

# Application logs (same as Ubuntu)
docker logs tiris-app-simple
```

## 🔄 Updates and Maintenance

### System Updates
```bash
# Update all packages
sudo dnf update -y

# Update security packages only
sudo dnf update --security -y

# Check for available updates
sudo dnf check-update

# Clean package cache
sudo dnf clean all
```

### Automatic Updates (configured by script)
```bash
# Check dnf-automatic status
sudo systemctl status dnf-automatic.timer

# View automatic update configuration
cat /etc/dnf/automatic.conf

# Manual trigger of automatic updates
sudo dnf-automatic
```

## 🆘 Common CentOS 9 Issues

### Issue: "No package docker available"
```bash
# Enable Docker CE repository
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo dnf install -y docker-ce
```

### Issue: Firewall blocking connections
```bash
# Check if firewalld is running
sudo systemctl status firewalld

# Temporarily disable for testing (NOT for production)
sudo systemctl stop firewalld

# Re-enable after testing
sudo systemctl start firewalld
```

### Issue: Container fails with "Permission denied"
```bash
# Check SELinux context
ls -laZ /opt/tiris/

# Fix SELinux contexts
sudo semanage fcontext -a -t container_file_t "/opt/tiris(/.*)?"
sudo restorecon -Rv /opt/tiris/
```

### Issue: "command not found" for Docker Compose
```bash
# CentOS 9 uses docker compose (not docker-compose)
docker compose version

# If you need the old docker-compose command
sudo curl -L "https://github.com/docker/compose/releases/download/v2.21.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

## 🎯 Performance Tuning (CentOS)

### System Limits
```bash
# Check current limits
ulimit -a

# Increase file descriptor limits (add to /etc/security/limits.conf)
echo "* soft nofile 65536" | sudo tee -a /etc/security/limits.conf
echo "* hard nofile 65536" | sudo tee -a /etc/security/limits.conf
```

### Network Tuning
```bash
# Optimize network parameters (add to /etc/sysctl.conf)
echo "net.core.somaxconn = 65535" | sudo tee -a /etc/sysctl.conf
echo "net.ipv4.tcp_max_syn_backlog = 65535" | sudo tee -a /etc/sysctl.conf
echo "net.core.netdev_max_backlog = 5000" | sudo tee -a /etc/sysctl.conf

# Apply changes
sudo sysctl -p
```

## 📚 Additional Resources

### CentOS 9 Documentation
- [CentOS 9 Documentation](https://docs.centos.org/en-US/stream/)
- [Docker on CentOS](https://docs.docker.com/engine/install/centos/)
- [Firewalld Documentation](https://firewalld.org/documentation/)

### Related Guides
- [Main Production Deployment](./PRODUCTION_DEPLOYMENT.md)
- [Operations Runbook](./OPERATIONS_RUNBOOK.md)
- [Quick Deploy Guide](../QUICK_DEPLOY.md)

---

**Note**: All the main deployment scripts automatically detect CentOS and handle the differences. This guide is for reference and troubleshooting specific to CentOS environments.