PVE相关的一些操作
---------------------------



### Failed to fetch https://enterprise.proxmox.com/debian/pve/dists/bullseye/InRelease  401  Unauthorized

```bash
cd /etc/apt/sources.list.d
mv pve-enterprise.list pve-enterprise.list.bak

cat > pve-enterprise.list <<EOF
deb http://download.proxmox.com/debian/pve $(cat pve-enterprise.list.bak | awk '{print $3}') pve-no-subscription
EOF

cd -
```



### No valid subscription

```bash
cd /usr/share/javascript/proxmox-widget-toolkit
cp proxmoxlib.js proxmoxlib.js.bak

// ...
void({ //Ext.Msg.show({
  title: gettext('No valid subscription'),
```



### 8006 -> 443

```bash
iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports 8006

# https://pve.proxmox.com/wiki/Certificate_Management
# https://www.willnet.net/index.php/archives/136/
```

