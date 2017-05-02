package main

import (
	"container/list"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/mapnificent/mapnificent_generator/mapnificent.pb"
	"github.com/nicolaspaton/gogtfs"
)

var (
	pathsString = flag.String("d", "", "Directories containing gtfs txt or zip files or zip file path (directories are traversed, multi coma separated: \"/here,/there\")")
	outputFile  = flag.String("o", "", "Output file")
	shouldLog   = flag.Bool("v", false, "Log to Stdout/err")
	needHelp    = flag.Bool("h", false, "Displays this help message...")
	feeds       map[string]*gtfs.Feed
)

func init() {
	feeds = make(map[string]*gtfs.Feed, 10)
}



func discoverGtfsPaths(path string) (results []string) {
	log.Println("discoverGtfsPaths: " + path)
	path = filepath.Clean(path)
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return
	}
	if fileInfo.IsDir() {
		file, err := os.Open(path)
		if err != nil {
			return
		}
		defer file.Close()
		fileInfos, err := file.Readdir(-1)
		if err != nil {
			return
		}
		requiredFiles := gtfs.RequiredFiles
		requiredCalendarFiles := gtfs.RequiredEitherCalendarFiles
		foundCalendar := false
		foundFiles := make([]string, 0, len(requiredFiles))
		for _, fi := range fileInfos {
			name := fi.Name()
			if fi.IsDir() {
				subdirectoryResults := discoverGtfsPaths(path + "/" + name)
				for _, newpath := range subdirectoryResults {
					results = append(results, newpath)
				}
			} else if filepath.Ext(name) == ".zip" {
				results = append(results, path+"/"+name)
			} else {
				for _, f := range requiredFiles {
					if name == f { // This loops a little too much but hey...
						foundFiles = append(foundFiles, f)
					}
				}
				if !foundCalendar {
					for _, f := range requiredCalendarFiles {
						if name == f {
							foundCalendar = true
						}
					}
				}

			}
		}
		if len(foundFiles) == len(requiredFiles) && foundCalendar {
			results = append(results, path)
		}

	} else {
		if filepath.Ext(path) == ".zip" {
			results = append(results, path)
		}
	}
	return
}

func GetNetwork(feeds map[string]*gtfs.Feed) *mapnificent.MapnificentNetwork {

	network := new(mapnificent.MapnificentNetwork)

	stationMap := make(map[string]uint)
	lineMap := make(map[string]*list.List)
	var name string

	for path, feed := range feeds {
		if name == "" {
			name = getNameFromPath(path)
			network.Cityid = proto.String(name)
		}

		for _, trip := range feed.Trips {
			tripHash := GetTripHash(trip)
			_, ok := lineMap[tripHash]
			if !ok {
				lineMap[tripHash] = list.New()
			}
			lineMap[tripHash].PushBack(trip)
		}
	}

	lineNo := 0
	for _, li := range lineMap {
		GetFrequencies(li, lineNo, network)

		trip := li.Front().Value.(*gtfs.Trip)

		agencyId := trip.Route.Agency.Id

		var lastStopArrival, lastStopDeparture uint
		var lastStop *mapnificent.MapnificentNetwork_Stop

		for _, stoptime := range trip.StopTimes {
			var mapnificentStop mapnificent.MapnificentNetwork_Stop

			stationName := fmt.Sprintf("%s_%s", agencyId, stoptime.Stop.Id)
			stopIndex, ok := stationMap[stationName]
			if !ok {
				mapnificentStop = mapnificent.MapnificentNetwork_Stop{
					Latitude:  proto.Float64(stoptime.Stop.Lat),
					Longitude: proto.Float64(stoptime.Stop.Lon),
				}
				network.Stops = append(network.Stops, &mapnificentStop)
				stopIndex = uint(len(network.Stops) - 1)
				stationMap[stationName] = stopIndex
			} else {
				mapnificentStop = *network.Stops[stopIndex]
			}

			if lastStop != nil {
				delta := stoptime.ArrivalTime - lastStopDeparture
				stayDelta := lastStopDeparture - lastStopArrival
				travelOption := mapnificent.MapnificentNetwork_Stop_TravelOption{
					Stop:       proto.Int32(int32(stopIndex)),
					TravelTime: proto.Int32(int32(delta)),
					StayTime:   proto.Int32(int32(stayDelta)),
					Line:       proto.String(trip.Id),
				}
				lastStop.TravelOptions = append(lastStop.TravelOptions, &travelOption)
			}
			lastStopArrival = stoptime.ArrivalTime
			lastStopDeparture = stoptime.DepartureTime
			lastStop = &mapnificentStop
		}

		lineNo++
	}
	return network
}

func GetFrequencies(trips *list.List, lineNo int, network *mapnificent.MapnificentNetwork) {
	var mapnificent_line mapnificent.MapnificentNetwork_Line
	var mapnificent_line_time mapnificent.MapnificentNetwork_Line_LineTime

	trip := trips.Front().Value.(*gtfs.Trip)
	// for trip := trips.Front(); trip != nil; trip = trip.Next() {

	mapnificent_line = mapnificent.MapnificentNetwork_Line{
		LineId: proto.String(trip.Id),
	}
	network.Lines = append(network.Lines, &mapnificent_line)
	lineIndex := uint(len(network.Lines) - 1)

	mapnificent_line_time = mapnificent.MapnificentNetwork_Line_LineTime{
		Interval: proto.Int32(int32(5)),
		Start:    proto.Int32(int32(0)),
		Stop:     proto.Int32(int32(24)),
		Weekday:  proto.Int32(int32(1 << 7)),
	}
	network.Lines[lineIndex].LineTimes = append(network.Lines[lineIndex].LineTimes, &mapnificent_line_time)
	// }
}

func GetTripHash(trip *gtfs.Trip) string {
	/* Gets a hash based on route and the trip stops */
	h := md5.New()
	io.WriteString(h, trip.Route.Id)
	for _, stoptime := range trip.StopTimes {
		io.WriteString(h, stoptime.Stop.Id)
	}
	return string(h.Sum(nil)[:md5.Size])
}

func getNameFromPath(path string) string {
	pathParts := strings.Split(path, "/")
	return pathParts[len(pathParts)-1]
}

func main() {
	flag.Parse()

	if *needHelp {
		flag.Usage()
		os.Exit(0)
	}

	pathsAll := strings.Split(*pathsString, ",")
	paths := make([]string, 0, len(pathsAll))

	for _, path := range pathsAll {
		if path != "" {
			// Why do I have to do this
			path = strings.Replace(path, "~", homeDir, 1)
			newpaths := discoverGtfsPaths(path)
			if len(newpaths) > 0 {
				for _, p := range newpaths {
					paths = append(paths, p)
				}
			}
		}
	}

	if !*shouldLog {
		devNull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0) // Shouldn't be an error
		defer devNull.Close()                                 // Useless, is it not?
		log.SetOutput(devNull)
	}

	log.SetPrefix("gtfs - ")

	channels := make([]chan bool, 0, len(paths))
	totalStopTimes := 0
	if len(paths) > 0 {
		for _, path := range paths[:] {
			log.Println(path)
			if path != "" {
				channel := make(chan bool)
				channels = append(channels, channel)
				go func(path string, ch chan bool) {
					log.Println("Started loading", path)
					feed, err := gtfs.NewFeed(path)
					if err != nil {
						log.Fatal(err)
					} else {
						feed.Load()
						feeds[path] = feed
					}
					totalStopTimes = totalStopTimes + feed.StopTimesCount
					channel <- true
				}(path, channel)
			}
		}
	}

	// Waiting for jobs to finnish
	for _, c := range channels {
		<-c
	}

	outfile, err := os.Create(*outputFile)
	if err != nil {
		return
	}

	network := GetNetwork(feeds)

	bytes, err := proto.Marshal(network)
	outfile.Write(bytes)
	outfile.Close()
}
