# Sonbonigisto
Voice message improving bot for Telegram group Nova Esperantujo.

It performs four tasks:
1. Compress sound (all from -27.5Db with ratio 3.0)
2. Truncate silence (all under -27.5Db to 33%)
3. Normalize (to -5Db)
4. Perform EQ change (for male voice)

It uses Audacity which requires DE for GUI.

To set up the system perform these commands.

## Install all dependencies
```shell
apt install tightvncserver dwm xfce4-terminal dbus-x11 ufw autocutsel figlet audacity pulseaudio
```

## Set up swap file and swappiness
```shell
swapoff /swap.img
rm /swap.img
fallocate -l 2G /swap.img
mkswap /swap.img
swapon /swap.img
sysctl vm.swappiness=200
nano /etc/sysctl.conf
```

## Set swappiness variable permanently
```shell
vm.swappiness=200
```

## Start up VNC server to get default config
```shell
vncserver :1
vncserver -kill :1
cd ~/.vnc
mv xstartup xstartup.bak
nano ~/.vnc/xstartup
```

## Add settings to config
```shell
#!/bin/bash
xrdb $HOME/.Xresources
autocutsel -fork
exec dwm
```

## Restart VNC server
```shell
chmod +x ~/.vnc/xstartup
vncserver :1
```

## Set up firewall
```shell
ufw default deny incoming
ufw default allow outgoing
ufw allow OpenSSH
ufw allow http
ufw allow https
ufw allow 5901/tcp
```

## Connect to VNC server via SSH
```shell
ssh -L 5901:127.0.0.1:5901 -C -N -l your-user your-server-IP-address
```
## Set up PuTTY to redirect SSH connection
Install PuTTY.

Go Connection > SSH > Tunnels...

Specify 5901 as source port and localhost:5901 as destination.

Add and apply.

## Connect to VNC server via TightVNC
Open TightVNC and connect to remote host localhost:5901.

Enter password specified earlier.

## Start Audacity with Telegram bot and log
```shell
./start.sh
./main 2>&1 | tee /var/tmp/sonbonigisto.tmp
```
