package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"./archivex" // "github.com/jhoonb/archivex"
	"./crc32"    // http://www.mrwaggel.be/post/generate-crc32-hash-of-a-file-in-golang-turorial/
	"github.com/jasonlvhit/gocron"
)

var currentBackup string
var zip *archivex.ZipFile

type configuration struct {
	Path     string `json:"path"`
	RunOnce  bool   `json:"runonce"`
	Interval int    `json:"interval"`
	Files    []struct {
		Name          string   `json:"name"`
		Path          string   `json:"path"`
		Except        []string `json:"except"`
		SkipCRCCheck  bool     `json:"skipCRCCheck"`
		KeepLastFiles int      `json:"keepLastFiles"`
	} `json:"files"`
}

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("Interrupt")
		incompleteBackup := currentBackup
		fmt.Println("[Backup] delete incomplete Backup:", incompleteBackup+".zip")

		err := zip.Close()
		if err != nil {
			panic(err)
		}

		err = os.Remove(incompleteBackup + ".zip")
		if err != nil {
			panic(err)
		}
		os.Exit(1)
	}()

	file, _ := os.Open("config.json")
	decoder := json.NewDecoder(file)
	config := configuration{}
	err := decoder.Decode(&config)

	if err != nil {
		fmt.Println("[Backup] Unable to load config.json, are you sure it's present? Error:", err.Error())
		os.Exit(1)
	}

	names := make([]string, 0)
	for _, json := range config.Files {
		names = append(names, json.Name)
	}

	fmt.Println("[Backup] Started up... Every:", config.Interval, "seconds. Preparing to archive:", strings.Join(names[:], ", "))

	backup := func() {
		fmt.Println()
		for _, json := range config.Files {
			zip = new(archivex.ZipFile)
			zipName := config.Path + json.Name + "@" + strconv.Itoa(int(time.Now().Unix()))
			zip.Create(zipName)
			currentBackup = zipName
			start := time.Now()
			zip.AddAll(json.Path, true, json.Except)
			zip.Close()
			currentBackup = ""
			fmt.Println("[Backup] Successfully archived:", zipName+".zip, elapsed", time.Since(start))
			// Alle Dateien mit dem gerade erstellten
			files, err := readDir(config.Path, json.Name+"@")
			if err != nil {
				panic(err)
			}
			if len(files) != 0 {
				if json.SkipCRCCheck == false {

					// überspringen da es nix gibt zum überprüfen gibt
					if len(files) < 2 {
						continue
					}

					// newhash vom gerade erstellten Backup
					newhash, err := crc32.Hash_file_crc32(zipName + ".zip")
					if err != nil {
						panic(err)
					}
					fmt.Println("[Debug] newhash", zipName+".zip", newhash)

					// oldhash vom letzten Backup
					oldhash, err := crc32.Hash_file_crc32(config.Path + files[len(files)-2])
					if err != nil {
						panic(err)
					}

					fmt.Println("[Debug] oldhash", config.Path+files[len(files)-2], oldhash)
					if oldhash == newhash {
						fmt.Println("[Backup] Same hash, delete ", zipName+".zip")
						err := os.Remove(zipName + ".zip")
						if err != nil {
							panic(err)
						}
						files = files[:len(files)-1]
					}
				}

				if json.KeepLastFiles != 0 {
					if len(files) > json.KeepLastFiles {
						filesDel := files[:len(files)-json.KeepLastFiles] // lösche die Dateien die behalten werden sollen
						for i := 0; i < len(filesDel); i++ {
							fmt.Println("[Backup] delete:", filesDel[i])
							err := os.Remove(config.Path + filesDel[i])
							if err != nil {
								panic(err)
							}
						}
					} else {
						fmt.Println("[Debug] not enough files")
					}
				}
			}

		}
	}

	Allstart := time.Now()
	backup()
	fmt.Println("[Backup] took", time.Since(Allstart))

	if config.RunOnce != true {
		s := gocron.NewScheduler()
		s.Every(uint64(config.Interval)).Seconds().Do(backup)
		<-s.Start()
	}
}

func readDir(root, limitation string) ([]string, error) {
	var files []string
	fileInfo, err := ioutil.ReadDir(root)
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		f := file.Name()

		if strings.HasPrefix(f, limitation) {
			files = append(files, f)
		}
	}
	return files, nil
}
