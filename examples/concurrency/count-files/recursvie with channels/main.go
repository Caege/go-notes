package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

func main() {

	logTxt, _ := os.OpenFile(
		"log_counter.txt",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)

	path := `C:\`

	fileCh := make(chan os.DirEntry)
	folderCh := make(chan string)

	var wg sync.WaitGroup

	wg.Add(1)
	go processFolder(path, folderCh, fileCh, &wg, logTxt)

	go func() {
		wg.Wait()
		fmt.Println("all folders processed")
		defer close(fileCh)
		defer close(folderCh)
	}()

	var fileCount int

	for folderCh != nil || fileCh != nil {
		select {
		case _, ok := <-fileCh:

			if !ok {
				fileCh = nil
				continue
			}

			fileCount++

		case folder, ok := <-folderCh:
			if !ok {
				folderCh = nil
				continue
			}

			go processFolder(folder, folderCh, fileCh, &wg, logTxt)
		}
	}

	logTxt.WriteString(fmt.Sprintf("file count : %d", fileCount))

	fmt.Println("\nfile count", fileCount)

}

func processFolder(dir string, folderCh chan string, fileCh chan os.DirEntry, wg *sync.WaitGroup, log *os.File) {

	defer wg.Done()
	defer fmt.Println("leaving", dir)
	defer log.WriteString(fmt.Sprintf("leaving dir : %s\n", dir))

	fmt.Println("found dir:", dir)
	log.WriteString(fmt.Sprintf("found dir : %s\n", dir))

	entries, err := os.ReadDir(dir)

	if err != nil {
		log.WriteString(fmt.Sprintf("error dir : %s\n", dir))
		fmt.Println("Error opening dir")
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			wg.Add(1)
			folderCh <- filepath.Join(dir, entry.Name())
		}

		if entry.Type().IsRegular() {
			fileCh <- entry
		}
	}

}
