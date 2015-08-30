/*
This program uses libpcap to capture MongoDB/TokuMX network traffic and generates a MySQL-style slow query log, that can then be used with pt-query-digest for workload analysis

IMPORTANT:

At this point this is alpha quality. The code is a mix of my own work, plus stuff I copied from facebookgo/dvara and gopacket/examples/pcapdump.
It may or may not get cleaned up and reorganized, depending on interest (from the community) and availability (from me/other maintainers).

*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unsafe"

	myutil "../util"
	"github.com/google/gopacket"
	"github.com/google/gopacket/examples/util"
	"github.com/google/gopacket/pcap"
	"gopkg.in/mgo.v2/bson"
)

var iface = flag.String("i", "eth0", "Interface to read packets from")
var fname = flag.String("r", "", "Filename to read from, overrides -i")
var snaplen = flag.Int("s", 65536, "Snap length (number of bytes max to read per packet")
var tstype = flag.String("timestamp_type", "", "Type of timestamps to use")
var promisc = flag.Bool("promisc", true, "Set promiscuous mode")
var verbose = 1

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

/*

From http://docs.mongodb.org/meta-driver/latest/legacy/mongodb-wire-protocol/

OP_UPDATE

struct OP_UPDATE {
    MsgHeader header;             // standard message header
    int32     ZERO;               // 0 - reserved for future use
    cstring   fullCollectionName; // "dbname.collectionname"
    int32     flags;              // bit vector. see below
    document  selector;           // the query to select the document
    document  update;             // specification of the update to perform
}

OP_INSERT

struct {
    MsgHeader header;             // standard message header
    int32     flags;              // bit vector - see below
    cstring   fullCollectionName; // "dbname.collectionname"
    document* documents;          // one or more documents to insert into the collection
}

OP_QUERY

struct OP_QUERY {
    MsgHeader header;                 // standard message header
    int32     flags;                  // bit vector of query options.  See below for details.
    cstring   fullCollectionName ;    // "dbname.collectionname"
    int32     numberToSkip;           // number of documents to skip
    int32     numberToReturn;         // number of documents to return
                                      //  in the first OP_REPLY batch
    document  query;                  // query object.  See below for details.
  [ document  returnFieldsSelector; ] // Optional. Selector indicating the fields
                                      //  to return.  See below for details.
}

so document is at offset 16 + 4 + N + 4 + 4

*/

// this map will store the start time for all requests. K:RequestId, v:StartTime
var startTimes = make(map[int32]time.Time)

// this map will store the query text (as I reconstructed/inferred it) for all requests. K:RequestId, v:Query
var queries = make(map[int32]string)

// my functions now

/*
struct OP_UPDATE {
    MsgHeader header;             // standard message header
    int32     ZERO;               // 0 - reserved for future use
    cstring   fullCollectionName; // "dbname.collectionname"
    int32     flags;              // bit vector. see below
    document  selector;           // the query to select the document
    document  update;             // specification of the update to perform
}

*/

func processUpdatePayload(data []byte, header messageHeader) (output string) {
	sub := data[20:]
	current := sub[0]
	docStartsAt := 0
	for i := 0; current != 0; i++ {
		current = sub[i]
		docStartsAt = i
	}
	collectionName := sub[0:docStartsAt]
	docStartsAt += 5 // ++ and then skip flags since I don't care about them right now
	mybson := sub[docStartsAt+8:]
	docEndsAt := mybson[0]
	bdoc := mybson[:docEndsAt]
	json := make(map[string]interface{})
	bson.Unmarshal(bdoc, json)
	if verbose > 2 {
		fmt.Print("Unmarshalled selector json: ")
		fmt.Println(json)
	}
	output = fmt.Sprintf("%v.update({", string(collectionName[:]))
	aux_output, _, _ := myutil.RecurseJsonMap(json)
	output += aux_output
	output += "},{"
	if verbose > 2 {
		fmt.Print("Selector: ")
		fmt.Print("mybson bytes: ")
		fmt.Println(mybson)
		fmt.Print("Document bytes:")
		fmt.Println(bdoc)
		fmt.Print("Document size in bytes: ")
		fmt.Println(unsafe.Sizeof(bdoc))
	}
	docEndsAt = sub[docEndsAt : docEndsAt+1][0]
	mybson = sub[docEndsAt+1:]
	bdoc = mybson[:docEndsAt]
	json = make(map[string]interface{})
	bson.Unmarshal(bdoc, json)
	if verbose > 2 {
		fmt.Print("Unmarshalled updater json: ")
		fmt.Println(json)
	}
	aux_output, _, _ = myutil.RecurseJsonMap(json)
	output += aux_output
	output += "});\n"
	return output
}

func processInsertPayload(data []byte, header messageHeader) (output string) {
	sub := data[20:]
	current := sub[0]
	docStartsAt := 0
	for i := 0; current != 0; i++ {
		current = sub[i]
		docStartsAt = i
	}
	collectionName := sub[0:docStartsAt]
	mybson := sub[docStartsAt+8:]
	docEndsAt := mybson[0]
	bdoc := mybson[:docEndsAt]
	json := make(map[string]interface{})
	bson.Unmarshal(bdoc, json)
	if verbose > 2 {
		fmt.Print("Unmarshalled selector json: ")
		fmt.Println(json)
	}
	output = fmt.Sprintf("%v.insert({", string(collectionName[:]))
	aux_output, _, _ := myutil.RecurseJsonMap(json)
	output += aux_output
	output += "});\n"
	return output
}

func processGetMorePayload(data []byte, header messageHeader) (output string) {
	//	if verbose > 2 {
	//		fmt.Println("Processing GetMore payload")
	//	}
	sub := data[20:]
	current := sub[0]
	docStartsAt := 0
	for i := 0; current != 0; i++ {
		current = sub[i]
		docStartsAt = i
	}
	collectionName := sub[0:docStartsAt]
	output = fmt.Sprintf("%v.getMore();\n", string(collectionName[:]))
	//if verbose > 2 {
	//fmt.Println(output)
	//}
	return output
}

func processQueryPayload(data []byte, header messageHeader) (output string) {
	sub := data[20:]
	current := sub[0]
	docStartsAt := 0
	for i := 0; current != 0; i++ {
		current = sub[i]
		docStartsAt = i
	}
	collectionName := sub[0:docStartsAt]
	docStartsAt++
	if verbose > 2 {
		fmt.Print("Raw data: ")
		fmt.Println(data)
		fmt.Printf("Querying collection %v\n", string(collectionName[:]))
	}
	if string(collectionName[len(collectionName)-4:len(collectionName)]) == "$cmd" {
		output = fmt.Sprintf("db.runCommand({")
	} else {
		output = fmt.Sprintf("%v.find({", string(collectionName[:]))
	}
	mybson := sub[docStartsAt+8:]
	docEndsAt := mybson[0]
	bdoc := mybson[:docEndsAt]
	json := make(map[string]interface{})
	bson.Unmarshal(bdoc, json)
	if verbose > 2 {
		fmt.Print("Unmarshalled json: ")
		fmt.Println(json)
	}
	if len(json) == 0 {
		output = fmt.Sprintf("%v.find();\n", string(collectionName[:]))
	} else {
		aux_output, _, _ := myutil.RecurseJsonMap(json)
		output += aux_output
		output += "});\n"
	}
	if verbose > 2 {
		fmt.Print("mybson bytes: ")
		fmt.Println(mybson)
		fmt.Print("Document bytes:")
		fmt.Println(bdoc)
		fmt.Print("Document size in bytes: ")
		fmt.Println(unsafe.Sizeof(bdoc))
	}
	return output
}

func processReplyPayload(data []byte, header messageHeader) (output float64) {
	var elapsed float64 = 0
	start, ok := startTimes[header.RequestID]
	if ok {
		elapsed = time.Since(start).Seconds()
		delete(startTimes, header.RequestID)
	}
	return elapsed
}

func dump(src gopacket.PacketDataSource) {
	var dec gopacket.Decoder
	var ok bool
	if dec, ok = gopacket.DecodersByLayerName["Ethernet"]; !ok {
		log.Fatalln("No decoder named", "Ethernet")
	}
	source := gopacket.NewPacketSource(src, dec)
	//source.Lazy = *lazy
	source.NoCopy = true
	for packet := range source.Packets() {
		//fmt.Println(packet.ApplicationLayer().Payload())
		al := packet.ApplicationLayer()
		/*
			defragger := ip4defrag.NewIPv4Defragmenter()
			in, err := defragger.DefragIPv4(packet.Layer(layers.LayerTypeIPv4))
			if err != nil {
				log.Fatalln(err)
			}
			if in == nil {
				fmt.Println("Got a fragment")
			} else {
				fmt.Println("Got a full packet or the last fragment of one")
			}
		*/
		if al != nil {
			/*
				json := make(map[string]interface{})
				//var mbson []bson.M
				bson.Unmarshal(al.Payload(), json)
				for k, v := range json {
					fmt.Println(k);
					fmt.Println(v);
				}
			*/
			payload := al.Payload()
			// IMPORTANT
			// This code is unsafe. It performs no check and will fail miserably if the packet is
			// not a mongo packet. Pass the proper 'port N' filter to pcap when invoking the program
			var header messageHeader
			header.MessageLength = getInt32(payload, 0)
			header.RequestID = getInt32(payload, 4)
			header.ResponseTo = getInt32(payload, 8)
			header.OpCode = OpCode(getInt32(payload, 12))
			startTimes[header.RequestID] = time.Now()
			if verbose > 2 {
				fmt.Println("OpCode: ", header.OpCode)
				fmt.Println("Captured packet")
				fmt.Printf("Captured packet (OpCode: %v)\n", header.OpCode)
			}
			switch header.OpCode {
			case OpQuery:
				queries[header.RequestID] = processQueryPayload(payload, header)
				if verbose > 2 {
					fmt.Printf("Saved Query for %v", header.RequestID)
				}
			case OpGetMore:
				queries[header.RequestID] = processGetMorePayload(payload, header)
				if verbose > 2 {
					fmt.Printf("Saved GetMore for %v", header.RequestID)
				}
			case OpReply:
				elapsed := processReplyPayload(payload, header)
				opInfo := make(myutil.OpInfo)
				opInfo["millis"] = fmt.Sprintf("%f", elapsed)
				opInfo["sent"] = fmt.Sprintf("%v", len(payload))
				fmt.Print(myutil.GetSlowQueryLogHeader(opInfo))
				query, ok := queries[header.ResponseTo]
				if ok {
					fmt.Print(query)
					delete(queries, header.ResponseTo)
				} else {
					if verbose > 1 {
						fmt.Printf("   Orphaned reply for %v\n", header.ResponseTo)
					}
				}
			case OpUpdate:
				queries[header.RequestID] = processUpdatePayload(payload, header)
			case OpInsert:
				queries[header.RequestID] = processInsertPayload(payload, header)
			default:
				if verbose > 1 {
					fmt.Println("Unimplemented Opcode ", header.OpCode)
				}
			}
		} // else {
		//	fmt.Println("empty? ", packet)
		//}
	}
}

// this main() is heavily inspired by / is a frankensteined version of https://github.com/google/gopacket/blob/master/examples/pcapdump/main.go

func main() {
	defer util.Run()()
	var handle *pcap.Handle
	var err error
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
		} else if err = inactive.SetPromisc(*promisc); err != nil {
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
		dump(handle)
	}
}
