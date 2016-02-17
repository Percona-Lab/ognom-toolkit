/*
This program uses libpcap to capture MongoDB/TokuMX network traffic and calculate request response time.

*/
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/examples/util"
	"github.com/google/gopacket/pcap"
	"github.com/prometheus/client_golang/prometheus"
)

var iface = flag.String("i", "eth0", "Interface to read packets from")
var fname = flag.String("r", "", "Filename to read from, overrides -i")
var snaplen = flag.Int("s", 65536, "Snap length (number of bytes max to read per packet")
var tstype = flag.String("timestamp_type", "", "Type of timestamps to use")
var port = flag.Int("p", 9119, "The http port to listen on")
var verbose = 1
var max = 0
var rtData = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "mongodb_response_time",
	Help:    "Response time for MongoDB operations",
	Buckets: prometheus.ExponentialBuckets(0.00000001, 2, 10),
})

// next code all copied from facebookgo/dvara

// OpCode allows identifying the type of operation:
//
// http://docs.mongodb.org/meta-driver/latest/legacy/mongodb-wire-protocol/#request-opcodes
type OpCode int32

// all data in the MongoDB wire protocol is little-endian.
// all the read/write functions below are little-endian.
func getInt32(b []byte, pos int) int32 {
	return (int32(b[pos+0])) |
		(int32(b[pos+1]) << 8) |
		(int32(b[pos+2]) << 16) |
		(int32(b[pos+3]) << 24)
}

func setInt32(b []byte, pos int, i int32) {
	b[pos] = byte(i)
	b[pos+1] = byte(i >> 8)
	b[pos+2] = byte(i >> 16)
	b[pos+3] = byte(i >> 24)
}

// The full set of known request op codes:
// http://docs.mongodb.org/meta-driver/latest/legacy/mongodb-wire-protocol/#request-opcodes
const (
	OpReply       = OpCode(1)
	OpMessage     = OpCode(1000)
	OpUpdate      = OpCode(2001)
	OpInsert      = OpCode(2002)
	Reserved      = OpCode(2003)
	OpQuery       = OpCode(2004)
	OpGetMore     = OpCode(2005)
	OpDelete      = OpCode(2006)
	OpKillCursors = OpCode(2007)
)

type messageHeader struct {
	// MessageLength is the total message size, including this header
	MessageLength int32
	// RequestID is the identifier for this miessage
	RequestID int32
	// ResponseTo is the RequestID of the message being responded to. used in DB responses
	ResponseTo int32
	// OpCode is the request type, see consts above.
	OpCode OpCode
}

// FromWire reads the wirebytes into this object
func (m *messageHeader) FromWire(b []byte) {
	m.MessageLength = getInt32(b, 0)
	m.RequestID = getInt32(b, 4)
	m.ResponseTo = getInt32(b, 8)
	m.OpCode = OpCode(getInt32(b, 12))
}

// this map will store the start time for all requests. K:RequestId, v:StartTime
var startTimes = make(map[int32]time.Time)

// my functions now

func processReplyPayload(data []byte, header messageHeader) (output float64) {
	var elapsed float64 = 0
	start, ok := startTimes[header.RequestID]
	if ok {
		elapsed = time.Since(start).Seconds()
		delete(startTimes, header.RequestID)
	}
	return elapsed
}

func startWebServer() {
	handler := prometheus.Handler()
	prometheus.MustRegister(rtData)
	strport := strconv.Itoa(*port)
	fmt.Println("Starting HTTP server on port " + strport)
	http.Handle("/metrics", handler)
	http.ListenAndServe(":"+strport, nil)
}

func process(src gopacket.PacketDataSource) {
	var dec gopacket.Decoder
	var ok bool
	if dec, ok = gopacket.DecodersByLayerName["Ethernet"]; !ok {
		log.Fatalln("No decoder named", "Ethernet")
	}
	source := gopacket.NewPacketSource(src, dec)
	//source.Lazy = *lazy
	source.NoCopy = true
	for packet := range source.Packets() {
		al := packet.ApplicationLayer()
		if al != nil {
			payload := al.Payload()
			if len(payload) < 16 {
				continue
			}
			//fmt.Println("len(payload) == %d", len(payload))
			// IMPORTANT
			// This code is unsafe. It performs no check and will fail miserably if the packet is
			// not a mongo packet. Pass the proper 'port N' filter to pcap when invoking the program
			var header messageHeader
			//fmt.Println(payload)
			header.MessageLength = getInt32(payload, 0)
			header.RequestID = getInt32(payload, 4)
			header.ResponseTo = getInt32(payload, 8)
			header.OpCode = OpCode(getInt32(payload, 12))
			startTimes[header.RequestID] = time.Now()
			//fmt.Printf("OpCode == %v\n", header.OpCode)
			switch header.OpCode {
			case OpReply:
				//fmt.Println("reply")
				rtData.Observe(processReplyPayload(payload, header))
				//fmt.Printf("%s,%20.10f\n", time.Now().Format("15:04:05"), rt)
			default:
			}
		}
	}
}

// this main() is heavily inspired by / is a frankensteined version of https://github.com/google/gopacket/blob/master/examples/pcapdump/main.go

func main() {
	defer util.Run()()
	var handle *pcap.Handle
	var err error
	go startWebServer()
	flag.Parse()
	if *fname != "" {
		if handle, err = pcap.OpenOffline(*fname); err != nil {
			log.Fatal("PCAP OpenOffline error:", err)
		}
	} else {
		// This is a little complicated because we want to allow all possible options
		// for creating the packet capture handle... instead of all this you can
		// just call pcap.OpenLive if you want a simple handle.
		inactive, err := pcap.NewInactiveHandle(*iface)
		if err != nil {
			log.Fatal("could not create: %v", err)
		}
		defer inactive.CleanUp()
		if err = inactive.SetSnapLen(*snaplen); err != nil {
			log.Fatal("could not set snap length: %v", err)
		} else if err = inactive.SetPromisc(true); err != nil {
			log.Fatal("could not set promisc mode: %v", err)
		} else if err = inactive.SetTimeout(time.Second); err != nil {
			log.Fatal("could not set timeout: %v", err)
		}
		if *tstype != "" {
			if t, err := pcap.TimestampSourceFromString(*tstype); err != nil {
				log.Fatalf("Supported timestamp types: %v", inactive.SupportedTimestamps())
			} else if err := inactive.SetTimestampSource(t); err != nil {
				log.Fatalf("Supported timestamp types: %v", inactive.SupportedTimestamps())
			}
		}
		if handle, err = inactive.Activate(); err != nil {
			log.Fatal("PCAP Activate error:", err)
		}
		defer handle.Close()
		if len(flag.Args()) > 0 {
			bpffilter := strings.Join(flag.Args(), " ")
			fmt.Fprintf(os.Stderr, "Using BPF filter %q\n", bpffilter)
			if err = handle.SetBPFFilter(bpffilter); err != nil {
				log.Fatal("BPF filter error:", err)
			}
		}
	}
	for {
		process(handle)
	}
}
