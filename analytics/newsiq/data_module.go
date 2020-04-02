package newsiq

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"cloud.google.com/go/storage"
	"github.com/golang/protobuf/proto"
	"github.com/mxmCherry/openrtb"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/prebid/prebid-server/analytics"
	"google.golang.org/api/iterator"
)

/*
type RequestType string

const (
	COOKIE_SYNC RequestType = "/cookie_sync"
	AUCTION     RequestType = "/openrtb2/auction"
	SETUID      RequestType = "/set_uid"
	AMP         RequestType = "/openrtb2/amp"
)
*/

/*
const STATUS = {
  BID_RECEIVED: 9,
  BID_WON: 10,
  BID_LOST: 11,
  NO_BID: 12,
  BID_TIMEOUT: 13
};

const MSG_TYPE = {
  Auction_Init: 101,
  Bid_Requested: 102,
  Bid_Response: 103,
  Bid_Timeout: 104,
  Bid_Won: 105,
  Bid_Lost: 106
};

let _topLevel = {};
let _queue = [];
let _sent = [];


Q: auction init, requested, response are all instances when we have to pass protobuf data to the data endpoint? Meaning we make 3 different calls for each prebid request?

A: prebid web create an auction instance, inside which it creates multi bidding instance for each bidder. For example, if the the configure says it need 3 bidders: apn, newsiq, rubicon, 3 bid instances will be created
auction init event triggers when the auction instance created
request event triggers when the requests to the 3 bid adapter fires
response event happens when the bid adapter returns bidding data
new messages
we need send protobuf data to the data endpoint to all 3 types of events
but not just 3 calls
auction init -- 1 call
request -- 3 calls (one for each bidder)
response - (0-3) calls
since response could fail, if all success, 3 calls, all fail 0 call
*/

type MsgType int

const (
	AuctionInit  MsgType = 101
	BidRequested MsgType = 102
	BidResponse  MsgType = 103
	BidTimeout   MsgType = 104
	BidWon       MsgType = 105
	BidLost      MsgType = 106
)

/**** PROD Variables ****/

const DebugLogging = true

const PrebidServerVersion = "0.97.0"
const newsIQDataEndpointDev = "https://newscorp-newsiq-dev.appspot.com/pb/"
const newsIQDataEndpointProd = "https://log.ncaudienceexchange.com/pb"

/**** PROD Variables ****/

func (d *DataLogger) LogAuctionObject(ao *analytics.AuctionObject) {
	if DebugLogging {
		fmt.Println("News IQ Module - LogAuctionObject")
	}
}
func (d *DataLogger) LogSetUIDObject(so *analytics.SetUIDObject) {
	if DebugLogging {
		fmt.Println("News IQ Module - LogSetUIDObject")
	}
}
func (d *DataLogger) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	if DebugLogging {
		fmt.Println("News IQ Module - LogCookieSyncObject")
	}
}
func (d *DataLogger) LogAmpObject(ao *analytics.AmpObject) {
	if DebugLogging {
		fmt.Println("News IQ Module - LogAmpObject")
	}
}
func (d *DataLogger) LogVideoObject(vo *analytics.VideoObject) {
	if DebugLogging {
		fmt.Println("News IQ Module - LogVideoObject")
	}
}

func currentTimestamp() uint64 {
	timeStamp := strings.ReplaceAll(time.Now().Format("01-02-2006"), "-", "")
	intTimeStamp, err := strconv.ParseUint(timeStamp, 10, 64)
	if err != nil {
		if DebugLogging {
			fmt.Println("Error: ", err)
		} // TODO: Log Error
		return 0
	}
	return intTimeStamp
}

func postData(prebidEventsData *LogPrebidEvents) {
	data, err := proto.Marshal(prebidEventsData)
	if err != nil {
		if DebugLogging {
			fmt.Println(err)
		} // TODO : Log this
	} else {
		if DebugLogging {
			fmt.Println("Protobuf Marshal success!")
		}
		_, err = http.Post(newsIQDataEndpointDev, "", bytes.NewBuffer(data))

		if err != nil {
			if DebugLogging {
				fmt.Println(err)
			} // TODO : Log this
		} else {
			if DebugLogging {
				fmt.Println("Protobuf POST request success!")
			}
		}

		if DebugLogging {
			// fmt.Println("RAW protobuf: ", data)
		}
	}
}

type DataLogger struct {
}

type DataTask struct {
	Request  *openrtb.BidRequest
	Response *openrtb.BidResponse
	Msg      MsgType
}

func NewDataLogger(filename string) analytics.PBSAnalyticsModule {
	if DebugLogging {
		fmt.Println("News IQ Module - NewDataLogger - ", filename)
	}
	return &DataLogger{}
}

func InitDataLogger() DataLogger {
	if DebugLogging {
		fmt.Println("News IQ Module - InitDataLogger")
	}

	// fmt.Println("ENVIRONMENT : ", Env)
	// if Env == "prod" {
	// 	bucketName = bucketNameProd
	// 	RunDataTaskService()
	// } else if Env == "dev" {
	// 	bucketName = bucketNameDev
	// 	RunDataTaskService()
	// }

	return DataLogger{}
}

var dataTaskChannel chan DataTask

/*
 TODO : Deprecated - no longer used - cleanup required
*/
func (d *DataLogger) StartDataTaskWorker() {
	dataTaskChannel = make(chan DataTask, 100)
	go dataTaskWorker(dataTaskChannel)
}

func (d *DataLogger) EnqueueDataTask(task DataTask) bool {
	if DebugLogging {
		fmt.Println("TEST : EnqueueDataTask(): ", dataTaskChannel)
	}
	select {
	case dataTaskChannel <- task:
		return true
	default:
		return false
	}
}

func dataTaskWorker(dataTaskChannel <-chan DataTask) {
	fmt.Println("Start loop")
	for task := range dataTaskChannel {
		fmt.Println("In loop: ", task.Msg)
		sendCollectorData(task.Request, task.Response, task.Msg)
	}
	fmt.Println("End loop")
}

func sendCollectorData(request *openrtb.BidRequest, response *openrtb.BidResponse, msg MsgType) {
	if DebugLogging {
		fmt.Println("News IQ Module - SendCollectorData() ", msg)
	}

	app := request.App
	site := request.Site

	var clientId uint64 = 0
	var clientDomain = ""

	if app != nil {
		if DebugLogging {
			fmt.Println("App type")
		}
		clientId, _ = strconv.ParseUint(app.ID, 10, 32)
		clientDomain = app.Domain
	} else if site != nil {
		if DebugLogging {
			fmt.Println("Site type")
		}
		clientId, _ = strconv.ParseUint(site.ID, 10, 32)
		clientDomain = site.Domain
	} else {
		if DebugLogging {
			fmt.Println("Client type key not found") // TODO : Log this
		}
	}

	adunitsArray := []*AdUnit{}
	prebidAuctionID := ""
	if response != nil {
		adunitsArray = generateAdUnits(request, response)
		prebidAuctionID = response.BidID // TODO : Is this correct?
	}

	auctionObj := Auction{
		Version:              PrebidServerVersion,
		AuctionInitTimestamp: currentTimestamp(), // TODO : Update all timestamps
		PrebidAuctionId:      prebidAuctionID,
		ConfiguredTimeoutMs:  30000, // 30 seconds
		MsgType:              uint32(msg),
		AdUnits:              adunitsArray,
	}
	auctionsArray := []*Auction{&auctionObj}

	device := request.Device
	// device.DeviceType // TODO : Include this?
	deviceString := device.Make + " " + device.Model + " " + device.HWV + " " + device.OS + " " + device.OSV
	prebidEventObj := &LogPrebidEvents{
		Timestamp:       currentTimestamp(),
		RemoteAddrMacro: request.Device.IP,
		UserAgentMacro:  request.Device.UA,
		// RefererUrl:      clientDomain, // TODO : Should be a page url
		SellerMemberId: uint32(clientId),
		Domain:         clientDomain,
		Device:         deviceString,
		Auctions:       auctionsArray,
		NewsId:         request.ID,
	}

	postData(prebidEventObj) // TODO : Remove old code
}

func generatePrebidLogData(request *openrtb.BidRequest, response *openrtb.BidResponse, msg MsgType) *LogPrebidEvents {
	if DebugLogging {
		fmt.Println("News IQ Module - generatePrebidLogData() ", msg)
		t := time.Now()
		ctx := context.TODO()
		defer TimeTrack(ctx, t, "GeneratePrebidLogData")
	}

	app := request.App
	site := request.Site

	var clientId uint64 = 0
	var clientDomain = ""

	if app != nil {
		if DebugLogging {
			fmt.Println("App type")
		}
		clientId, _ = strconv.ParseUint(app.ID, 10, 32)
		clientDomain = app.Domain
	} else if site != nil {
		if DebugLogging {
			fmt.Println("Site type")
		}
		clientId, _ = strconv.ParseUint(site.ID, 10, 32)
		clientDomain = site.Domain
	} else {
		if DebugLogging {
			fmt.Println("Client type key not found") // TODO : Log this
		}
	}

	adunitsArray := []*AdUnit{}
	prebidAuctionID := ""
	if response != nil {
		adunitsArray = generateAdUnits(request, response)
		prebidAuctionID = response.BidID // TODO : Is this correct?
	}

	auctionObj := Auction{
		Version:              PrebidServerVersion,
		AuctionInitTimestamp: currentTimestamp(), // TODO : Update all timestamps
		PrebidAuctionId:      prebidAuctionID,
		ConfiguredTimeoutMs:  30000, // 30 seconds
		MsgType:              uint32(msg),
		AdUnits:              adunitsArray,
	}
	auctionsArray := []*Auction{&auctionObj}

	device := request.Device
	// device.DeviceType // TODO : Include this?
	deviceString := device.Make + " " + device.Model + " " + device.HWV + " " + device.OS + " " + device.OSV
	prebidEventObj := &LogPrebidEvents{
		Timestamp:       currentTimestamp(),
		RemoteAddrMacro: request.Device.IP,
		UserAgentMacro:  request.Device.UA,
		// RefererUrl:      clientDomain, // TODO : Should be a page url
		SellerMemberId: uint32(clientId),
		Domain:         clientDomain,
		Device:         deviceString,
		Auctions:       auctionsArray,
		NewsId:         request.ID,
	}

	return prebidEventObj
}

func generateBidsArray(identifier string, response *openrtb.BidResponse) []*Bid {
	bidsArray := []*Bid{}
	seatBid := response.SeatBid
	if seatBid != nil {
		for _, seatbidObj := range seatBid {
			for _, bidObj := range seatbidObj.Bid {
				if bidObj.ID == identifier {
					creativeObj := &Creative{
						CreativeId: bidObj.CrID,
						Width:      uint32(bidObj.W),
						Height:     uint32(bidObj.H),
						Brand:      bidObj.Bundle,
					}

					bidResponseId := BidResponse
					if len(bidObj.NURL) > 0 { // Win
						bidResponseId = BidWon
					} else if len(bidObj.LURL) > 0 { // Lost
						bidResponseId = BidLost
					}
					bidTrackingObj := Bid{
						BidId:             fmt.Sprint(bidResponseId),
						Price:             bidObj.Price,
						BidderCode:        seatbidObj.Seat,
						BidderAdUnitId:    bidObj.ImpID,
						RequestTimestamp:  10212011,
						ResponseTimestamp: 02132011,
						// StatusCode:        uint32(ao.Status),  TODO : Bring this back in
						Source:   "",
						Creative: creativeObj,
					}
					bidsArray = append(bidsArray, &bidTrackingObj)
				}
			}
		}
	}
	return bidsArray
}

func generateAdUnits(request *openrtb.BidRequest, response *openrtb.BidResponse) []*AdUnit {
	adunitsArray := []*AdUnit{}
	for _, impressionObj := range request.Imp {
		paramsArray := []*Param{}
		if impressionObj.Banner != nil {
			paramsArray = append(paramsArray,
				&Param{
					Key:   "adunit_type",
					Value: "banner",
				}, &Param{
					Key:   "width",
					Value: fmt.Sprint(impressionObj.Banner.W),
				}, &Param{
					Key:   "height",
					Value: fmt.Sprint(impressionObj.Banner.H),
				}, &Param{
					Key:   "pos",
					Value: fmt.Sprint(impressionObj.Banner.Pos),
				}, &Param{
					Key:   "id",
					Value: fmt.Sprint(impressionObj.Banner.ID),
				}, &Param{
					Key:   "extension",
					Value: fmt.Sprint(impressionObj.Banner.Ext),
				})
		} else if impressionObj.Native != nil {
			paramsArray = append(paramsArray,
				&Param{
					Key:   "adunit_type",
					Value: "native",
				}, &Param{
					Key:   "request",
					Value: impressionObj.Native.Request,
				}, &Param{
					Key:   "version",
					Value: impressionObj.Native.Ver,
				}, &Param{
					Key:   "extension",
					Value: fmt.Sprint(impressionObj.Native.Ext),
				})
		} else if impressionObj.Video != nil {
			paramsArray = append(paramsArray,
				&Param{
					Key:   "adunit_type",
					Value: "video",
				}, &Param{
					Key:   "width",
					Value: fmt.Sprint(impressionObj.Video.W),
				}, &Param{
					Key:   "height",
					Value: fmt.Sprint(impressionObj.Video.H),
				}, &Param{
					Key:   "pos",
					Value: fmt.Sprint(impressionObj.Video.Pos),
				}, &Param{
					Key:   "min_duration",
					Value: fmt.Sprint(impressionObj.Video.MinDuration),
				}, &Param{
					Key:   "max_duration",
					Value: fmt.Sprint(impressionObj.Video.MaxDuration),
				}, &Param{
					Key:   "max_duration",
					Value: fmt.Sprint(impressionObj.Video.MaxDuration),
				}, &Param{
					Key:   "extension",
					Value: fmt.Sprint(impressionObj.Video.Ext),
				})
		} else if impressionObj.Audio != nil {
			paramsArray = append(paramsArray,
				&Param{
					Key:   "adunit_type",
					Value: "audio",
				}, &Param{
					Key:   "min_duration",
					Value: fmt.Sprint(impressionObj.Audio.MinDuration),
				}, &Param{
					Key:   "max_duration",
					Value: fmt.Sprint(impressionObj.Audio.MaxDuration),
				}, &Param{
					Key:   "sequence",
					Value: fmt.Sprint(impressionObj.Audio.Sequence),
				}, &Param{
					Key:   "extension",
					Value: fmt.Sprint(impressionObj.Audio.Ext),
				})
		} else {
			if DebugLogging {
				fmt.Println("Impression object type missing") // TODO : Log this
			}
		}

		bidsArray := generateBidsArray(impressionObj.ID, response)
		adunitObj := AdUnit{
			AdUnitCode: impressionObj.ID,
			Bids:       bidsArray,
			Params:     paramsArray,
		}
		adunitsArray = append(adunitsArray, &adunitObj)
	}

	return adunitsArray
}

/***** Google Storage Variables *****/

var bucketNameProd, bucketNameDev, bucketName string = "newscorp-newsiq-stage-bq", "newscorp-newsiq-dev-bq", ""
var Env = os.Getenv("PREBID_ENV")

var NewLineBytes = len([]byte("\n"))

var validPrebidRecordsKeyName = []byte("{\n\"validPrebidRecords\":")
var invalidPrebidRecordsKeyName = []byte(",\n\"invalidPrebidRecords\":")
var jsonMsgEnder = []byte("\n}")

/***** Core *****/

type GcsGzFileRoller struct {
	io.Writer
	client        *storage.Client
	bucket        string
	ctx           context.Context
	fileNameTmplt string
	instanceId    string

	dateStr         string
	hourStr         string
	dateHourStr     string
	isStreaming     bool
	filePathAndName string
	cBytesWritten   uint64
	uBytesWritten   uint64
	recordsWritten  uint64

	fi *storage.Writer
	gf *gzip.Writer
	fw *bufio.Writer
}

func (f *GcsGzFileRoller) Write(buf []byte) (int, error) {
	n, err := f.fi.Write(buf)
	if DebugLogging {
		if err != nil {
			fmt.Println("TEST : Write() Error - ", err, " | context: ", f.ctx)
		} else {
			fmt.Println("TEST : Write() Success - ", f.ctx)
		}
	}
	atomic.AddUint64(&f.cBytesWritten, uint64(n))
	return n, err
}

func (f *GcsGzFileRoller) IncrementByteCount(size int) *GcsGzFileRoller {
	f.uBytesWritten += uint64(size)
	return f
}

func (f *GcsGzFileRoller) NextGZ() *GcsGzFileRoller {

	f.cBytesWritten = 0
	f.uBytesWritten = 0
	f.recordsWritten = 0

	tm := time.Now().UTC()
	f.dateStr = fmt.Sprintf("%d%02d%02d", tm.Year(), tm.Month(), tm.Day())
	f.hourStr = fmt.Sprintf("%02d", tm.Hour())
	f.dateHourStr = f.dateStr + "-" + f.hourStr
	fileName := fmt.Sprintf(f.fileNameTmplt, f.dateStr, f.dateHourStr, f.instanceId, tm.Unix())
	f.filePathAndName = fileName

	if DebugLogging {
		fmt.Println("TEST : NextGZ()", f.ctx, " Bucket: ", f.bucket, " Filename: ", fileName, " Instance: ", f.instanceId)
	}

	fi := f.client.Bucket(f.bucket).Object(fileName).NewWriter(f.ctx)
	f.fi = fi
	gf := gzip.NewWriter(f)
	nameArray := strings.Split(fileName, "/")
	gf.Name = nameArray[len(nameArray)-1]
	fw := bufio.NewWriter(gf)
	f.gf = gf
	f.fw = fw
	f.isStreaming = true
	return f
}

func (f *GcsGzFileRoller) WriteGZ(logData *LogPrebidEvents, brf *GcsGzFileRoller) *GcsGzFileRoller {
	if DebugLogging {
		fmt.Println("TEST : WriteGZ() - ", f.ctx)
	}
	var jsonMsg []byte

	// payload := &serialize.LogPrebidEvents{}
	// if err := proto.Unmarshal(b, payload); err != nil {
	// 	if DebugLogging {
	// 		fmt.Println("News IQ Module - SendCollectorData() ", msg)
	// 	}
	// 	log.Warningf(f.ctx, "can't deserialize protobuf payload: %v", err)
	// 	return f
	// }
	var theArray [3]string
	theArray[0] = "India"  // Assign a value to the first element
	theArray[1] = "Canada" // Assign a value to the second element
	theArray[2] = "Japan"
	if rslt, err := ffjson.Marshal(theArray); err == nil {
		jsonMsg = rslt
	} else {
		if DebugLogging {
			fmt.Println(f.ctx, "can't json marshal LogPrebidEvents data payload: %v", err)
		}
		// log.Errorf(f.ctx, "can't json marshal protobuf payload: %v", err) // TODO : Update
		return f
	}

	// call function to verify and validate timestamp and prebid_auction_id columns

	/* TODO : Verify and add logic back in ???

	var writeRecord bool = VerifyFields(logData)

	if writeRecord == false {
		// log.Errorf(f.ctx, "Field verification: %v", "Field verification has failed either for timestamp or prebid_auction_id") // TODO : Update
		LogJsonMsg(brf, jsonMsg)
		return f
	}
	*/

	LogJsonMsg(f, jsonMsg)

	return f
}

func (f *GcsGzFileRoller) CloseCurrentGZ() *GcsGzFileRoller {
	if DebugLogging {
		fmt.Println("TEST : CloseCurrentGZ() - ", f.ctx)
	}
	f.fw.Flush()
	f.gf.Close()
	f.fi.Close()
	f.isStreaming = false
	return f
}

func (f *GcsGzFileRoller) GetStats() []byte {
	rslt, _ := ffjson.Marshal(&map[string]string{
		"isStreaming":              strconv.FormatBool(f.isStreaming),
		"bucket":                   f.bucket,
		"instanceId":               f.instanceId,
		"filePathAndName":          f.filePathAndName,
		"compressedBytesWritten":   strconv.FormatUint(f.cBytesWritten, 10),
		"uncompressedBytesWritten": strconv.FormatUint(f.uBytesWritten, 10),
		"recordsWritten":           strconv.FormatUint(f.recordsWritten, 10),
	})
	return rslt
}

func VerifyFields(prebidData *LogPrebidEvents) bool {
	var writeData bool = true
	timestampValue := strconv.FormatUint((*prebidData).Timestamp, 10)
	t, err := strconv.ParseInt(timestampValue, 10, 64)
	if err != nil {
		fmt.Println("timestamp parsing error : ", err)
		fmt.Println("Maximum value allowed for timestamp string is : ", t)
		return false
	}

	t = 0 // to avoid Go's error for unused variables

	for _, element := range (*prebidData).Auctions {
		if (*element).PrebidAuctionId == "" {
			writeData = false
			break
		}
	}
	return writeData
}

func LogJsonMsg(f *GcsGzFileRoller, msg []byte) {
	if DebugLogging {
		fmt.Println("TEST : LogJsonMsg")
	}
	if !f.isStreaming {
		f.NextGZ()
	}
	tm := time.Now().UTC()
	dateHourStr := fmt.Sprintf("%d%02d%02d-%02d", tm.Year(), tm.Month(), tm.Day(), tm.Hour())
	if f.dateHourStr != dateHourStr || f.cBytesWritten >= 524288000 {
		// log.Infof(f.ctx, "closing '%s' stream to roll to next hour or due to size (>=500MB)", f.filePathAndName) // TODO : Update
		f.CloseCurrentGZ().NextGZ()
		if DebugLogging {
			fmt.Println("TEST : LogJsonMsg - Closing GZ")
		}
	}
	(f.fw).Write(msg)
	(f.fw).WriteString("TESTING \n") // TEST :
	f.IncrementByteCount(len(msg) + NewLineBytes)
	f.recordsWritten++
	if DebugLogging {
		fmt.Println("TEST : LogData() records written = ", f.recordsWritten)
	}
}

func LogData(c <-chan DataTask, f *GcsGzFileRoller, brf *GcsGzFileRoller) {
	if DebugLogging {
		fmt.Println("TEST : LogData() in NewsIQ Data Module")
	}
	defer f.CloseCurrentGZ()
	defer f.client.Close()
	defer brf.CloseCurrentGZ()
	defer brf.client.Close()
	for {
		select {
		case incoming := <-c:
			logData := generatePrebidLogData(incoming.Request, incoming.Response, incoming.Msg)
			f.WriteGZ(logData, brf)
		case <-time.After(time.Second * 60): //<-close stream after 60 seconds of inactivity
			if f.isStreaming {
				f.CloseCurrentGZ()
				if DebugLogging {
					fmt.Println("TEST : LogData() closing stream due to inactivity: ", f.filePathAndName)
				}
				// log.Infof(f.ctx, "closing '%s' stream due to inactivity", f.filePathAndName) // TODO : Update
			}
			if brf.isStreaming {
				brf.CloseCurrentGZ()
				if DebugLogging {
					fmt.Println("TEST : LogData() closing stream due to inactivity: ", brf.filePathAndName)
				}
				// log.Infof(f.ctx, "closing '%s' stream due to inactivity", brf.filePathAndName) // TODO : Update
			}
		}
	}
}

// func listBuckets(ctx context) {
// 	var buckets []string
// 	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
// 	defer cancel()
// 	client, err := storage.NewClient(ctx)
// it := client.Buckets(ctx, projectID)
// for {
// 	battrs, err := it.Next()
// 	if err == iterator.Done {
// 		break
// 	}
// 	if err != nil {
// 		fmt.Println("TEST : Error bucket name - ", err)
// 		return
// 	}
// 	fmt.Println("TEST : bucket name - ", battrs.Name)
// 	buckets = append(buckets, battrs.Name)
// }
// }

// func listBucketObjects(ctx Context) {
// 	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
// 	defer cancel()
// 	client, err := storage.NewClient(ctx)
// 	it := client.Bucket(f.bucket).Objects(ctx, nil)
// 	for {
// 		attrs, err := it.Next()
// 		if err == iterator.Done {
// 			break
// 		}
// 		if err != nil {
// 			fmt.Println("TEST : Error bucket objects - ", err)
// 			return
// 		}
// 		fmt.Println("TEST : bucket object name - ", attrs.Name)
// 	}
// }

/***** Data Service *****/

func (d *DataLogger) RunDataTaskService() {
	fmt.Println("ENVIRONMENT : ", Env)
	if Env == "prod" {
		bucketName = bucketNameProd
	} else if Env == "dev" {
		bucketName = bucketNameDev
	}
	ctx := context.Background()

	// listBuckets(ctx)

	// listBucketObjects(ctx)

	// ctx := appengine.BackgroundContext() // TODO : remove old code
	// logCh := make(chan []byte, 1111) // TODO : remove old code
	client, err := storage.NewClient(ctx)
	bucket := bucketName
	if err != nil {
		fmt.Println(ctx, "failed to create client: %v", err)
		// log.Errorf(ctx, "failed to create client: %v", err) // TODO : Remove old code
		return
	}

	it := client.Bucket(bucket).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println("TEST : Item name Error ", err)
			return
		}
		fmt.Println("TEST : Item name - ", attrs.Name)
	}

	roller := &GcsGzFileRoller{
		client:        client,
		bucket:        bucket,
		ctx:           ctx,
		fileNameTmplt: "newsiq-prebidserver-logs/dt=%s/%s-%s-%v.json.gz",
		instanceId:    os.Getenv("PBS_INSTANCE_ID"),
	}

	badRecordsRoller := &GcsGzFileRoller{
		client:        client,
		bucket:        bucket,
		ctx:           ctx,
		fileNameTmplt: "newsiq-prebidserver-badrecords-logs/dt=%s/%s-%s-%v.json.gz",
		instanceId:    os.Getenv("PBS_INSTANCE_ID"),
	}

	// go LogData(logCh, roller, badRecordsRoller) // TODO : Remove old code
	dataTaskChannel = make(chan DataTask, 100)
	go LogData(dataTaskChannel, roller, badRecordsRoller)
}

func TimeTrack(ctx context.Context, start time.Time, name string) {
	elapsed := time.Since(start)
	took := elapsed.Nanoseconds() / 1000
	if took > 500 {
		fmt.Println(ctx, "%s took %dms", name, took)
		// log.Warningf(ctx, "%s took %dms", name, took) // TODO : Remove old code
	}
}

func (d *DataLogger) EnqueuePrebidDataTask(task DataTask) bool {
	if DebugLogging {
		fmt.Println("TEST : EnqueuePrebidDataTask(): ", task)
	}
	// ctx := appengine.NewContext(r) // TODO : Update this context
	/* TODO : Moving all over to generatePrebidLogData
	t := time.Now()
	ctx := context.TODO()
	defer TimeTrack(ctx, t, "EnqueuePrebidDataTask")
	*/
	// responseData, err := ioutil.ReadAll(r.Body)
	// if err != nil {
	// 	log.Warningf(ctx, "%v", err)
	// 	return
	// }
	// c <- responseData
	select {
	case dataTaskChannel <- task:
		return true
	default:
		return false
	}
}
