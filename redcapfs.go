// get REDCap connected into a fuse file system

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HaukeBartsch/redcapfs/nodefsC"
	"github.com/HaukeBartsch/redcapfs/utils"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/howeyc/gopass"
)

var participants []map[string]string
var instruments []map[string]string
var formEventMapping []map[string]string
var mountPoint string
var tokens map[string][]string

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
		if ext != ".json" && ext != ".csv" && ext != ".xlsx" {
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
			} else if ext == ".csv" {
				go func() { utils.WriteAsCsv(instruments, p) }()
			} else if ext == ".xlsx" {
				go func() { utils.WriteAsExcel(instruments, p) }()
			} else {
				fmt.Println("Error: this file extension is not known")
			}
			return
		}

		// lets see if this is a variable or an instrument
		inst := ""
		meas := ""
		for _, entry := range instruments {
			//fmt.Println("test an instrument now: ", entry["form_name"], variable)
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
				in := utils.GetInstrument(inst, tokens)
				dd := utils.GetDataDictionary([]string{inst}, tokens)
				in = filterByDate(in, p)
				ddname := strings.TrimSuffix(p, filepath.Ext(p)) + "_datadictionary" + filepath.Ext(p)
				//in = filterBySite(in, p)
				if ext == ".json" {
					utils.WriteAsJson(in, p)
					utils.WriteAsJson(dd, ddname)
				} else if ext == ".csv" {
					utils.WriteAsCsv(in, p)
					utils.WriteAsCsv(dd, ddname)
				} else if ext == ".xlsx" {
					utils.WriteAsExcel(in, p)
					utils.WriteAsExcel(dd, ddname)
				} else {
					fmt.Println("Error: unknown format to write")
				}
			}()
		}
		if meas != "" {
			go func() {
				me := utils.GetMeasure(meas, tokens)
				me = filterByDate(me, p)
				//me = filterBySite(me, p)
				if ext == ".json" {
					utils.WriteAsJson(me, p)
				} else if ext == ".csv" {
					utils.WriteAsCsv(me, p)
				} else if ext == ".xlsx" {
					utils.WriteAsExcel(me, p)
				} else {
					fmt.Println("Error: unknown format to write")
				}
			}()
		}
		if (meas == "") && (inst == "") {
			fmt.Println("Error: value is neither variable nor instrument ", variable)
		}
	} else if what == "MKDIR" {
		// create a directory, could be event name
		fmt.Println("asked to create a directory, could be event name...")
		l := strings.Split(path, "/")
		event := l[len(l)-1]
		dir, err := filepath.Abs(mountPoint)
		if err != nil {
			log.Fatal(err)
		}
		p := fmt.Sprintf("%s/%s", dir, path)
		for _, v := range formEventMapping {
			if v["unique_event_name"] == event {
				/* filename := fmt.Sprintf("%s/%s", p, v["form"])
				fmt.Println("Found the event, write now form:", filename)
				_, err := os.Stat(p)
				if os.IsNotExist(err) {
					fmt.Println("Error: directory does not exist yet")
				}
				go func(filename string) {
					time.Sleep(500 * time.Millisecond)
					fmt.Println("Found the event, write now form:", filename)
					var nodata []map[string]string
					// we might have to delay the creation of the file until the directory is really created...
					utils.WriteAsJson(nodata, filename)
				}(filename) */
				// ok, we found unique_event_name, create its json representation underneath
				time.Sleep(500 * time.Millisecond)
				go func(form string) {
					me := utils.GetInstrument(form, tokens)
					me = filterByDate(me, p)
					//me = filterBySite(me, p)
					fn := fmt.Sprintf("%s/%s.json", p, form)
					utils.WriteAsJson(me, fn)
				}(v["form"])
			}
		}
	}
}

func filterBySite(what []map[string]string, path string) []map[string]string {
	fmt.Println("filter by sites now")
	// find out if we have a date field in the path
	l := strings.Split(path, "/")
	var whatNew []map[string]string

	// find all sites that we have access to
	sites := make(map[string]bool, 0)
	for _, entry := range participants {
		for k, v := range entry {
			fmt.Println("entry has: ", k, " and ", v)
		}
		site := strings.Split(entry["redcap_data_access_group"], "_de")
		if len(site) > 0 {
			sites[strings.ToUpper(site[0])] = true
		} else {
			fmt.Println("did not find data access group")
		}
	}
	for k := range sites {
		fmt.Println("sites are:", k)
	}
	foundSiteString := false
	for _, v := range l {
		if v == "" {
			continue
		}
		if sites[strings.ToUpper(v)] {
			foundSiteString = true
			fmt.Println("Found a site string, filter by this site", v)
			for _, entry := range what {
				p := entry["id_redcap"]
				for _, ps := range participants {
					if ps["id_redcap"] == p {
						tsite := strings.Split(ps["redcap_data_access_group"], "_de")
						if len(tsite) == 0 {
							continue
						}
						if strings.ToUpper(tsite[0]) == strings.ToUpper(v) {
							whatNew = append(whatNew, entry)
						}
					} else {
						fmt.Println("Skip this entry", ps["id_redcap"], ". redcap_data_access_group ", ps["redcap_data_access_group"], "is not site", v)
					}
				}
			}
		}
	}
	if foundSiteString == false {
		whatNew = what
	}
	return whatNew
}

func filterByDate(what []map[string]string, path string) []map[string]string {
	// find out if we have a date field in the path
	l := strings.Split(path, "/")
	var whatNew []map[string]string
	foundTimeString := false
	for _, v := range l {
		if v == "" {
			continue
		}
		//fmt.Println("Test if", v, "is a time string")
		t, err := time.Parse("Jan 2006", v)
		if err == nil {
			foundTimeString = true
			// fmt.Println("Found a time string, filter by this date (same month as baseline)")
			for _, entry := range what {
				p := entry["id_redcap"]
				for _, ps := range participants {
					if ps["id_redcap"] == p {
						// found the participant now look at its baseline date
						td, err := time.Parse("2006-01-02 15:04", ps["cp_timestamp_v2"])
						if err != nil {
							fmt.Println("Could not parse baseline date from", ps["cp_timestamp_v2"])
							continue
						}
						if (t.Month() == td.Month()) && (t.Year() == td.Year()) {
							whatNew = append(whatNew, entry)
						} else {
							fmt.Println("Skip this entry", ps["id_redcap"], ". Date ", ps["cp_timestamp_v2"], "is not in requested range", v)
						}
					}
				}
			}
		}
	}
	if foundTimeString == false {
		whatNew = what
	}
	return whatNew
}

func main() {
	// Scans the arg list and sets up flags
	debug := flag.Bool("debug", false, "print debugging messages.")
	addToken := flag.String("addToken", "", "add a <REDCap token>")
	showToken := flag.Bool("showToken", false, "show existing token")
	clearAllTokens := flag.Bool("clearAllToken", false, "remove stored token")
	setREDCap := flag.String("setREDCapURL", "https://abcd-rc.ucsd.edu/redcap/api/", "set the REDCap URL")
	flag.Parse()

	// get the pass-phrase
	fmt.Printf("This is a secured access. Provide your pass phrase: ")
	pw, err := gopass.GetPasswd() // Silent
	if err != nil {
		fmt.Println("Error: could not read pass-phrase")
		panic(err)
	}
	// try to save this as a global variable
	tokens = utils.TokenStoreGet(string(pw[:]))
	if *showToken == true {
		str, err := json.Marshal(tokens)
		if err != nil {
			fmt.Println("Error, could not convert token to string")
			panic(err)
		}
		fmt.Println("Tokens are: \n", string(str))
		os.Exit(0)
	}
	if *addToken != "" {
		tokens["accessTokens"] = append(tokens["accessTokens"], *addToken)
		utils.TokenStorePut(string(pw[:]), tokens)
		os.Exit(0)
	}
	if *clearAllTokens == true {
		utils.TokenStoreRemove(string(pw[:]))
		os.Exit(0)
	}
	if *setREDCap != "" {
		tokens["REDCapURL"] = append(tokens["REDCapURL"], *setREDCap)
	}

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

	// get values we might need later (or not)
	participants = utils.GetParticipantsBySite(tokens)
	instruments = utils.GetInstruments(tokens)
	formEventMapping = utils.GetFormEventMapping(tokens)

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
