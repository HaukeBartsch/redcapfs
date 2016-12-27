// get REDCap connected into a fuse file system

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/HaukeBartsch/redcapfs/nodefsC"
	"github.com/HaukeBartsch/redcapfs/utils"
	"github.com/hanwen/go-fuse/fuse"
)

var participants []string
var instruments []map[string]string
var formEventMapping []map[string]string
var mountPoint string

// For example: If the user creates a folder with a given name, can be use that folders name
// to populate the directory created?
// As an example one could create a directory with the name of an instrument. We would like to
// see the data from that instrument inside the folder.
// Another example is one could create a folder with a measure and if that measure is categorical
// we would see files with the data for that measure for all participants (data.json).
// Can we get stages of filters done this way as well? For example a directory tree could
// represent an AND/OR. Maybe its easier to start on the highest level with all data and
// only reduce the data in subsequent levels.
func somethingHappened(path string, what string) {
	//fmt.Println("something happend on the file system, got ", path, what, "\n")

	// ok we have access now to the path and to the instrument + participants
	// we should do something if we see a CREATE event (== filename)
	if (what == "CREATE") || (what == "RENAME") {
		// we create a file in this directory
		// lets use the filename and query REDCap with that variable
		ext := filepath.Ext(path)
		if ext != ".json" && ext != ".csv" {
			return
		}
		path2 := strings.TrimSuffix(path, filepath.Ext(path))
		l := strings.Split(path2, "/")
		variable := l[len(l)-1]
		//fmt.Println("Var:", variable)

		dir, err := filepath.Abs(mountPoint)
		if err != nil {
			log.Fatal(err)
		}
		p := fmt.Sprintf("%s/%s", dir, path)
		fmt.Println("path we will write: ", p)

		if variable == "DataDictionary" {
			if ext == ".json" {
				go func() { utils.WriteAsJson(instruments, p) }()
			} else {
				go func() { utils.WriteAsCsv(instruments, p) }()
			}
			return
		}

		// lets see if this is a variable or an instrument
		inst := ""
		meas := ""
		for _, entry := range instruments {
			if entry["form_name"] == variable {
				inst = variable
				break
			}
			if entry["field_name"] == variable {
				meas = variable
				break
			}
		}

		if inst != "" {
			go func() {
				in := utils.GetInstrument(inst)
				if ext == ".json" {
					utils.WriteAsJson(in, p)
				} else {
					utils.WriteAsCsv(in, p)
				}
			}()
		}
		if meas != "" {
			go func() {
				me := utils.GetMeasure(meas)
				if ext == ".json" {
					utils.WriteAsJson(me, p)
				} else {
					utils.WriteAsCsv(me, p)
				}
			}()
		}
	} else if what == "MKDIR" {
		// create a directory, could be event name
		fmt.Println("asked to create a directory, could be event name")
		l := strings.Split(path, "/")
		event := l[len(l)-1]
		dir, err := filepath.Abs(mountPoint)
		if err != nil {
			log.Fatal(err)
		}
		p := fmt.Sprintf("%s/%s", dir, path)
		for _, v := range formEventMapping {
			if v["unique_event_name"] == event {
				// ok, we found unique_event_name, create its json representation underneath
				go func(form string) {
					me := utils.GetInstrument(form)
					fn := fmt.Sprintf("%s/%s.json", p, form)
					utils.WriteAsJson(me, fn)
				}(v["form"])
			}
		}
	}
}

func main() {
	// Scans the arg list and sets up flags
	debug := flag.Bool("debug", false, "print debugging messages.")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Println("usage: main MOUNTPOINT BACKING-PREFIX")
		os.Exit(2)
	}

	mountPoint = flag.Arg(0)
	prefix := "meme"
	if flag.NArg() == 2 {
		prefix = flag.Arg(1)
	}
	root := nodefsC.NewFSNodeFSRoot(prefix, somethingHappened)
	conn := nodefsC.NewFileSystemConnector(root, nil)
	server, err := fuse.NewServer(conn.RawFS(), mountPoint, &fuse.MountOptions{
		Debug: *debug,
	})
	if err != nil {
		fmt.Printf("Mount fail: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Mounted!")

	// set the current list of tokens for access
	utils.Tokens["UCSD"] = "9560341DB5CD569629990DD4BF8D5947"

	// get values we might need later (or not)
	participants = utils.GetParticipantsBySite()
	instruments = utils.GetInstruments()
	formEventMapping = utils.GetFormEventMapping()

	go func() {
		dir, err := filepath.Abs(mountPoint)
		if err != nil {
			log.Fatal(err)
		}
		p := fmt.Sprintf("%s/%s", dir, "DataDictionary.json")

		fmt.Println("Writing data dictionary to ", p)
		utils.WriteAsJson(instruments, p)
	}()
	go func() {
		dir, err := filepath.Abs(mountPoint)
		if err != nil {
			log.Fatal(err)
		}
		p := fmt.Sprintf("%s/%s", dir, "EventMapping.json")

		fmt.Println("Writing event mapping to ", p)
		utils.WriteAsJson(formEventMapping, p)
	}()

	server.Serve()
}
