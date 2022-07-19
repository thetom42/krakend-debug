#!/bin/sh

###### Krakend logging to influxdb
# krakend access
influx user create -n krakend

###### Krakend logging to influxdb
# set up a compatibility layer for carmen-api-gateway, i.e. krakend
# krakend v2 still uses influxdb v1 access methods

# add influxdb v1 compatible user / password access
id=$(influx bucket list -o carmen-crm -name krakend | tail -1 | cut -f1)
influx v1 auth create --read-bucket "$id" --write-bucket "$id" --username krakend --password "krakend123"
influx v1 auth list

# allow influxdb v1 compatible access to krakend bucket
influx v1 dbrp create --bucket-id "$id" --db krakend --rp krakend --default
influx v1 dbrp list -o carmen-crm

###### Add some predefined dashboards
# to create dashboards from a running influxdb - pod:
# - inside pod execute:
#   influx export all --filter=resourceKind=Dashboard > /tmp/all_dashboards.yml
# - copy from pod to local file
#   kubectl cp influx-uniq-0:/tmp/all_dashboards.yml k8s/base/config/all_dashboards.yml

if [ -r /config/all_dashboards.yml ]; then
    influx apply -o carmen-crm --file /config/all_dashboards.yml -q --force yes
fi

###### Krakend payload logging

# add bucket for payload logging
influx bucket create -n logpayload -o carmen-crm -r 30d
buckid=$(influx auth create -o carmen-crm --write-bucket logpayload | cut -f1)
influx auth create -o carmen-crm --write-bucket $buckid -u krakend
# - get token required for logging i.e. systemteam-secret.INFLUX_TOKEN
#   influx auth list | grep $(influx bucket list | grep logpayload | cut -f1) | cut -f10
