# ulog2queue

Sends ulogd / netfilter json logs to elasticsearch backend (logstash replacement for your firewall)
(originally ulog2queue name comes from the fact that it was to be used to send to a redis queue)

* Support as primary backend: elasticsearch, redis, rabbitmq (not complete yet)
* Support as backup backend: disk,redis,rabbitmq (not complete yet)

WARNING: work in progress, but it is used be me on multiple firewalls generating each about 7000 events/sec.

Documentation is a bit sparse for now, but you can open issues or contact @42wim on twitter if things are not clear.

## use cases
* You've got a linux firewall with netfilter and you're running ulogd to handle your logging.
* You don't want to run logstash on your firewall because it's java and rather slow.
* You've got CPU cycles to spare on your firewall.
* You want to do geoip of the IP addresses of your firewall (IPv4 AND IPv6).
* You want ISO8601 @timestamps in ES like logstash gives you.
* You want one binary.

Then ulog2queue is for you!

The common use case would be pushing the data into ES backend and using the disk as backup (when ES is too slow or when ES nodes are down). See Examples

More info about ulogd and json logging: [regit blog](https://home.regit.org/2014/02/using-ulogd-and-json-output/)

## requirements
* ulogd 2.0.5 
* geoip database (https://dev.maxmind.com/geoip/geoip2/geolite2/ and download GeoLite2 City)
* elasticsearch cluster

## building
Make sure you have [Go](https://golang.org/doc/install) properly installed.

ulog2queue uses the [gb tool](http://getgb.io) to manage dependencies and producing builds.


```
git clone https://github.com/42wim/ulog2queue.git
cd ulog2queue
gb build all
```

You should now have ulog2queue binary in the bin directory:

```
$ ls bin/
ulog2queue
```

## usage
```
Usage of ./ulog2queue:
  -backup="": name of backup backend (disk/redis/rabbit)
  -conf="ulog2queue.cfg": config file
  -debug=false: enable debug
  -primary="": name of primary backend (es/redis/rabbit)
  -queue="icts-p-netconf-2": name of queue
  -tail="": file to tail
```

## config
ulog2queue looks for ulog2queue.cfg in current directory.  

Look at ulog2queue.cfg.sample for an example

## example

The common use case would be pushing the data into ES backend and using the disk as backup (when ES is too slow or when ES nodes are down)

### Step 1 - ulogd
Get ulogd with json support configured (see ulogd.conf.sample) for an example.
Create a named pipe /var/log/ulogd.json
```
# mknod /var/log/ulogd.json p
```
We're going to tail /var/log/ulogd.json.

### Step 2 - geoip2 database
Download geoip2 from (https://dev.maxmind.com/geoip/geoip2/geolite2/ and download GeoLite2 City),unzip and put into /usr/share/ulog2queue/GeoLite2-City.mmdb

### Step 3 - create /etc/ulog2queue.cfg
```
[backend "disk"]
#pick a path that has enough storage. Every message is ~1KB.
URI="/var/log/ulogd.backup"

[backend "es"]
#URL of your ES cluster
URI="http://http-query.fw-log.service.consul:9200"
#index name, uses golang time format, must match template in ulog2queue-es-mapping.json
index="ulog2queue-fw-2006.01.02"
#uses ES bulk API, batches 5000 messages
bulk=5000
#5 concurrent workers
workers=5

[general]
primary="es"
backup="disk"
geoip2db="/usr/share/ulog2queue/GeoLite2-City.mmdb"
tailfile="/var/log/ulogd.json"
buffer=10000
```

### Step 4 - create elasticsearch mapping
```
$ curl -XPUT http://fw-log.service.consul:9200/_template/json-log -d "@ulog2queue-es-mapping.json"
```

### Step 5 - systemd unit
```
# cp ulog2queue.service /etc/systemd/system/ulog2queue.service && systemctl daemon-reload
```

### Step 6 - start ulog2queue and ulogd
```
# systemctl start ulog2queue
# systemctl start ulogd
```
