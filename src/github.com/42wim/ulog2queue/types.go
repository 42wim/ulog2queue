package main

type Context struct {
	lines           chan *string //input from ulog
	parsedLines     chan *[]byte //parsed (timestamp/geoip) - primary backend
	backupLines     chan *[]byte //backup backend
	parsedRate      chan int
	backupRate      chan int
	restoreStart    chan string
	restoreDone     chan bool
	restoreFilename chan string
	backupRateInt   int
	buffering       bool
	cfg             *Config
}

type esGeoIPLocation struct {
	Lon *float64 `json:"lon"`
	Lat *float64 `json:"lat"`
}

type esGeoIP struct {
	Longitude      *float64         `json:"longitude"`
	Latitude       *float64         `json:"latitude"`
	CountryName    string           `json:"country_name,omitempty"`
	Timezone       *string          `json:"timezone,omitempty"`
	ContinentCode  *string          `json:"continent_code,omitempty"`
	CityName       string           `json:"city_name,omitempty"`
	CountryCode2   *string          `json:"country_code2,omitempty"`
	RealRegionName *string          `json:"real_region_name,omitempty"`
	RegionName     *string          `json:"region_name,omitempty"`
	IP             *string          `json:"ip,omitempty"`
	Location       *esGeoIPLocation `json:"location,omitempty"`
	Coordinates    *esGeoIPLocation `json:"coordinates,omitempty"`
}

type nf struct {
	Ahespspi         *int    `json:"ahesp.spi,omitempty"`
	Arpdaddrstr      *string `json:"arp.daddr.str,omitempty"`
	Arpdhwaddr       *int    `json:"arp.dhwaddr,omitempty"`
	Arphwtype        *int    `json:"arp.hwtype,omitempty"`
	Arpoperation     *int    `json:"arp.operation,omitempty"`
	Arpprotocoltype  *int    `json:"arp.protocoltype,omitempty"`
	Arpsaddrstr      *string `json:"arp.saddr.str,omitempty"`
	Arpshwaddr       *int    `json:"arp.shwaddr,omitempty"`
	Ctevent          *int    `json:"ct.event,omitempty"`
	Ctid             *int    `json:"ct.id,omitempty"`
	Ctmark           *int    `json:"ct.mark,omitempty"`
	Flowendsec       *int    `json:"flow.end.sec,omitempty"`
	Flowendusec      *int    `json:"flow.end.usec,omitempty"`
	Flowstartsec     *int    `json:"flow.start.sec,omitempty"`
	Flowstartusec    *int    `json:"flow.start.usec,omitempty"`
	Icmpcode         *int    `json:"icmp.code,omitempty"`
	Icmpcsum         *int    `json:"icmp.csum,omitempty"`
	Icmpechoid       *int    `json:"icmp.echoid,omitempty"`
	Icmpechoseq      *int    `json:"icmp.echoseq,omitempty"`
	Icmpfragmtu      *int    `json:"icmp.fragmtu,omitempty"`
	Icmpgateway      *int    `json:"icmp.gateway,omitempty"`
	Icmptype         *int    `json:"icmp.type,omitempty"`
	Icmpv6code       *int    `json:"icmpv6.code,omitempty"`
	Icmpv6csum       *int    `json:"icmpv6.csum,omitempty"`
	Icmpv6echoid     *int    `json:"icmpv6.echoid,omitempty"`
	Icmpv6echoseq    *int    `json:"icmpv6.echoseq,omitempty"`
	Icmpv6type       *int    `json:"icmpv6.type,omitempty"`
	Ip6flowlabel     *int    `json:"ip6.flowlabel,omitempty"`
	Ip6fragid        *int    `json:"ip6.fragid,omitempty"`
	Ip6fragoff       *int    `json:"ip6.fragoff,omitempty"`
	Ip6hoplimit      *int    `json:"ip6.hoplimit,omitempty"`
	Ip6nexthdr       *int    `json:"ip6.nexthdr,omitempty"`
	Ip6payloadlen    *int    `json:"ip6.payloadlen,omitempty"`
	Ip6priority      *int    `json:"ip6.priority,omitempty"`
	Ipcsum           *int    `json:"ip.csum,omitempty"`
	Ipdaddrstr       *string `json:"ip.daddr.str,omitempty"`
	Ipfragoff        *int    `json:"ip.fragoff,omitempty"`
	Ipid             *int    `json:"ip.id,omitempty"`
	Ipihl            *int    `json:"ip.ihl,omitempty"`
	Ipprotocol       *int    `json:"ip.protocol,omitempty"`
	Ipsaddrstr       *string `json:"ip.saddr.str,omitempty"`
	Iptos            *int    `json:"ip.tos,omitempty"`
	Iptotlen         *int    `json:"ip.totlen,omitempty"`
	Ipttl            *int    `json:"ip.ttl,omitempty"`
	Macdaddrstr      *string `json:"mac.daddr.str,omitempty"`
	Macsaddrstr      *string `json:"mac.saddr.str,omitempty"`
	Macstr           *string `json:"mac.str,omitempty"`
	Nufwappname      *string `json:"nufw.app.name,omitempty"`
	Nufwosname       *string `json:"nufw.os.name,omitempty"`
	Nufwosrel        *string `json:"nufw.os.rel,omitempty"`
	Nufwosvers       *string `json:"nufw.os.vers,omitempty"`
	Nufwuserid       *int    `json:"nufw.user.id,omitempty"`
	Nufwusername     *string `json:"nufw.user.name,omitempty"`
	Oobfamily        *int    `json:"oob.family,omitempty"`
	Oobgid           *int    `json:"oob.gid,omitempty"`
	Oobhook          *int    `json:"oob.hook,omitempty"`
	Oobifindexin     *int    `json:"oob.ifindex_in,omitempty"`
	Oobifindexout    *int    `json:"oob.ifindex_out,omitempty"`
	Oobin            *string `json:"oob.in,omitempty"`
	Oobmark          *int    `json:"oob.mark"`
	Oobout           *string `json:"oob.out,omitempty"`
	Oobprefix        *string `json:"oob.prefix,omitempty"`
	Oobprotocol      *int    `json:"oob.protocol,omitempty"`
	Oobseqglobal     *int    `json:"oob.seq.global,omitempty"`
	Oobseqlocal      *int    `json:"oob.seq.local,omitempty"`
	Oobtimesec       *int    `json:"oob.time.sec,omitempty"`
	Oobtimeusec      *int    `json:"oob.time.usec,omitempty"`
	Oobuid           *int    `json:"oob.uid,omitempty"`
	Origipdaddrstr   *string `json:"orig.ip.daddr.str,omitempty"`
	Origipprotocol   *int    `json:"orig.ip.protocol,omitempty"`
	Origipsaddrstr   *string `json:"orig.ip.saddr.str,omitempty"`
	Origl4dport      *int    `json:"orig.l4.dport,omitempty"`
	Origl4sport      *int    `json:"orig.l4.sport,omitempty"`
	Origrawpktcount  *int    `json:"orig.raw.pktcount,omitempty"`
	Origrawpktlen    *int    `json:"orig.raw.pktlen,omitempty"`
	Print            *string `json:"print,omitempty"`
	Pwsniffpass      *string `json:"pwsniff.pass,omitempty"`
	Pwsniffuser      *string `json:"pwsniff.user,omitempty"`
	Rawlabel         *int    `json:"raw.label,omitempty"`
	Rawmacaddrlen    *int    `json:"raw.mac.addrlen,omitempty"`
	Rawmac           *int    `json:"raw.mac,omitempty"`
	Rawmaclen        *int    `json:"raw.mac_len,omitempty"`
	Rawpktcount      *int    `json:"raw.pktcount,omitempty"`
	Rawpkt           *int    `json:"raw.pkt,omitempty"`
	Rawpktlen        *int    `json:"raw.pktlen,omitempty"`
	Rawtype          *int    `json:"raw.type,omitempty"`
	Replyipdaddrstr  *string `json:"reply.ip.daddr.str,omitempty"`
	Replyipprotocol  *int    `json:"reply.ip.protocol,omitempty"`
	Replyipsaddrstr  *string `json:"reply.ip.saddr.str,omitempty"`
	Replyl4dport     *int    `json:"reply.l4.dport,omitempty"`
	Replyl4sport     *int    `json:"reply.l4.sport,omitempty"`
	Replyrawpktcount *int    `json:"reply.raw.pktcount,omitempty"`
	Replyrawpktlen   *int    `json:"reply.raw.pktlen,omitempty"`
	Sctpcsum         *int    `json:"sctp.csum,omitempty"`
	Sctpdport        *int    `json:"sctp.dport,omitempty"`
	Sctpsport        *int    `json:"sctp.sport,omitempty"`
	Sumbytes         *int    `json:"sum.bytes,omitempty"`
	Sumname          *string `json:"sum.name,omitempty"`
	Sumpkts          *int    `json:"sum.pkts,omitempty"`
	Tcpack           *int    `json:"tcp.ack,omitempty"`
	Tcpackseq        *int    `json:"tcp.ackseq,omitempty"`
	Tcpcsum          *int    `json:"tcp.csum,omitempty"`
	Tcpdport         *int    `json:"tcp.dport,omitempty"`
	Tcpfin           *int    `json:"tcp.fin,omitempty"`
	Tcpoffset        *int    `json:"tcp.offset,omitempty"`
	Tcppsh           *int    `json:"tcp.psh,omitempty"`
	Tcpreserved      *int    `json:"tcp.reserved,omitempty"`
	Tcpres1          *int    `json:"tcp.res1,omitempty"`
	Tcpres2          *int    `json:"tcp.res2,omitempty"`
	Tcprst           *int    `json:"tcp.rst,omitempty"`
	Tcpseq           *int    `json:"tcp.seq,omitempty"`
	Tcpsport         *int    `json:"tcp.sport,omitempty"`
	Tcpsyn           *int    `json:"tcp.syn,omitempty"`
	Tcpurg           *int    `json:"tcp.urg,omitempty"`
	Tcpurgp          *int    `json:"tcp.urgp,omitempty"`
	Tcpwindow        *int    `json:"tcp.window,omitempty"`
	Udpcsum          *int    `json:"udp.csum,omitempty"`
	Udpdport         *int    `json:"udp.dport,omitempty"`
	Udplen           *int    `json:"udp.len,omitempty"`
	Udpsport         *int    `json:"udp.sport,omitempty"`

	Srcport    *int    `json:"src_port,omitempty"`
	Srcip      *string `json:"src_ip,omitempty"`
	Destport   *int    `json:"dest_port,omitempty"`
	Destip     *string `json:"dest_ip,omitempty"`
	Dvc        *string `json:"dvc,omitempty"`
	Timestamp  *string `json:"timestamp,omitempty"`
	Ltimestamp string  `json:"@timestamp,omitempty"`
	Action     *string `json:"action,omitempty"`

	GeoIP esGeoIP `json:"geoip"`
}
