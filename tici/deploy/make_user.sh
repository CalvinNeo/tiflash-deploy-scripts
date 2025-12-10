#!/bin/bash

USER=tidb
PUBKEY=`cat ~/.ssh/id_rsa.pub`

useradd -m -s /bin/bash $USER

# 创建 .ssh 目录并设置权限
mkdir -p /home/$USER/.ssh
chmod 700 /home/$USER/.ssh
chown $USER:$USER /home/$USER/.ssh

echo "$PUBKEY" > /home/$USER/.ssh/authorized_keys
chmod 600 /home/$USER/.ssh/authorized_keys
chown $USER:$USER /home/$USER/.ssh/authorized_keys

ssh-keygen -A

# 配置 sshd_config，允许公钥登录
sed -i 's/^#*PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config 2>/dev/null || echo "PermitRootLogin yes" >> /etc/ssh/sshd_config
sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config 2>/dev/null || echo "PasswordAuthentication no" >> /etc/ssh/sshd_config
sed -i 's/^#*PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config 2>/dev/null || echo "PubkeyAuthentication yes" >> /etc/ssh/sshd_config

mkdir -p /var/run/sshd

# 后台启动 SSH 服务
/usr/sbin/sshd

echo "tidb ALL=(ALL) NOPASSWD:ALL" | sudo tee /etc/sudoers.d/tidb
sudo chmod 440 /etc/sudoers.d/tidb