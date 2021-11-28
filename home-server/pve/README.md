PVE相关的一些操作
---------------------------

网络拓扑图（在线链接：https://www.processon.com/view/link/61a38768e401fd48c0b5e50c）

![离线拓扑图](../\assets\pve-server.jpg)





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
  
systemctl restart pveproxy.service
```



### 8006 -> 443

```bash
iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports 8006

# https://pve.proxmox.com/wiki/Certificate_Management
# https://www.willnet.net/index.php/archives/136/
```



### 直通

```bash
# https://foxi.buduanwang.vip/yj/561.html/

cat >> /etc/modules <<EOF
vfio
vfio_iommu_type1
vfio_pci
vfio_virqfd
EOF
```

