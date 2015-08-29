package main

import (
	"encoding/json"
	"fmt"
	"github.com/SaidinWoT/gulf/glob"
	set "github.com/deckarep/golang-set"
	"gopkg.in/fsnotify.v1"
	"log"
	"os"
	"os/exec"
	"time"
)

type Entry struct {
	Name  string
	Files []string
	Run   []string
}

var config []Entry

var configMap map[string]Entry

func readConfig() {
	config = make([]Entry, 0)
	configMap = make(map[string]Entry)
	file, _ := os.Open("conf.json")
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&config)
	if err != nil {
		fmt.Println("error:", err)
	}
	for id, entry := range config {
		configMap[entry.Name] = config[id]
	}
	toRun = set.NewSet()
}

var toRun set.Set

func runEntry(name string) {
	entry, ok := configMap[name]
	if !ok {
		fmt.Println("Couldn't find entry ", name)
		return
	}
	fmt.Println("=====")
	fmt.Println(name)
	for _, cmd := range entry.Run {

		fmt.Println("\033[31m", cmd, "\033[0m")
		proc := exec.Command("bash", "-c", cmd)
		proc.Stdout = os.Stdout
		proc.Stderr = os.Stderr
		err := proc.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
	fmt.Println("=====")
}

func run() {
	for entryNameI := range toRun.Iter() {
		entryName, ok := entryNameI.(string)
		if !ok {
			fmt.Println("Error converting entryname:", entryNameI)
			continue
		}
		go runEntry(entryName)
	}
}
func filesUpdated(set set.Set) {
	fmt.Println("Updated:")
	toRun.Clear()
	for file := range set.Iter() {
		fmt.Println(file)
		filestr, ok := file.(string)
		if !ok {
			fmt.Println("Error converting to string:\n", file)
			continue
		}
		for _, entry := range config {
			for _, pattern := range entry.Files {
				if toRun.Contains(entry.Name) {
					break
				}
				match, err := glob.Match(pattern, filestr)
				if err != nil {
					fmt.Println(err)
					continue
				}
				if match {
					toRun.Add(entry.Name)
				}
			}
		}
	}
	run()
}

func unused() {
	fileEntry := make(map[string][]Entry)
	entries := config
	for _, entry := range entries {
		fmt.Println(entry.Name)
		for _, file := range entry.Files {
			fmt.Println("  ", file)
			registeredEntries, ok := fileEntry[file]
			if !ok {
				registeredEntries = make([]Entry, 1)
				fileEntry[file] = entries
			}
			fileEntry[file] = append(registeredEntries, entry)
		}
	}

	file := "blah"
	var matches []string
	matches, err := glob.Glob(file)
	if err != nil {
		fmt.Print("Err", err)
		return
	}
	for matchid, match := range matches {
		fmt.Println("    ", matchid, match)
	}
}
func main() {
	readConfig()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		run := false
		delay := 1000 * time.Millisecond
		timer := time.NewTimer(delay)
		updated := set.NewSet()
		for {
			select {
			case <-timer.C:
				if run {
					run = false
					filesUpdated(updated)
					updated.Clear()
				}
			case event := <-watcher.Events:
				timer.Reset(delay)
				//if event.Op&fsnotify.Write == fsnotify.Write {
				run = true
				updated.Add(event.Name)
				//}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	wd, err := os.Getwd()
	if err == nil {
		fmt.Println("Monitoring", wd)
		err = watcher.Add(wd)
	}
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
