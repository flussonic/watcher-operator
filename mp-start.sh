#!/bin/bash

set -ex

if [ -f env ]; then
    set -a
    source ./env
    set +a
fi

if [ -z "$LICENSE_KEY" ]; then
    read -p "Enter Flussonic license key: "  LICENSE_KEY
fi


multipass launch --name watcher --cpus 1 --memory 4096M --disk 5G lts
multipass exec watcher -- sudo /bin/sh -c 'apt remove -y snapd'
multipass exec watcher -- sudo /bin/sh -c 'curl -sfL https://get.k3s.io | sh -'
# kubectl label nodes streamer flussonic.com/streamer=true

# token=$(multipass exec watcher sudo cat /var/lib/rancher/k3s/server/node-token)
plane_ip=$(multipass info watcher | grep -i ip | awk '{print $2}')
rm -f k3s.yaml
multipass exec watcher sudo cat /etc/rancher/k3s/k3s.yaml |sed "s/127.0.0.1/${plane_ip}/" > k3s.yaml
chmod 0400 k3s.yaml
export KUBECONFIG=`pwd`/k3s.yaml

kubectl create secret generic flussonic-license \
    --from-literal=license_key="${LICENSE_KEY}" \
    --from-literal=login="${LOGIN}" \
    --from-literal=pass="${PASS}"



# kubectl apply -f ./docs/operator.yaml
# kubectl apply -f config/samples/media_v1alpha1_watcher.yaml


echo "Watcher ready: http://${plane_ip}"
