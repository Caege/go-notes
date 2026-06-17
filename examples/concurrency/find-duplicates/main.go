package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"sync/atomic"
)

type Results struct {
	newDirs []string
}

type SafeMap struct {
	mu sync.Mutex
	sm map[int64][]string
}

func main() {
	jobs := make(chan string)
	results := make(chan Results)

	var fileCounter atomic.Int64
	active := 0
	var pending []string


    //replace this path your own path
	pending = append(pending, `V:\`)

	workerCount := 4

	sizeMap := SafeMap{
		sm: make(map[int64][]string),
	}

	for range workerCount {
		go func() {
			for job := range jobs {
				entries, err := os.ReadDir(job)

				if err != nil {
					results <- Results{}
					continue
				}
				var newFolders []string
				for _, entry := range entries {

					fullpath := filepath.Join(job, entry.Name())

					if entry.Type().IsRegular() && filepath.Ext(entry.Name()) == ".mp4" {

						info, err := entry.Info()

						size := info.Size()

						if err != nil {
							fmt.Println("Error getting the file info")
							continue
						}

						sizeMap.mu.Lock()
						sizeMap.sm[size] = append(sizeMap.sm[size], fullpath)
						sizeMap.mu.Unlock()

						fileCounter.Add(1)
					}

					if entry.IsDir() {
						fmt.Println("discovered : ", entry.Name())
						newFolders = append(newFolders, fullpath)
					}
				}

				results <- Results{newDirs: newFolders}
			}

		}()
	}

	isDone := false

	for !isDone {
		for active < workerCount && len(pending) > 0 {
			firstJob := pending[0]
			pending = pending[1:]

			active++

			jobs <- firstJob

		}

		result := <-results
		active--
		pending = append(pending, result.newDirs...)

		if active == 0 && len(pending) == 0 {

			close(jobs)
			fmt.Println("file count", fileCounter.Load())
			isDone = true

		}
	}

	sizeMap.mu.Lock()
	defer sizeMap.mu.Unlock()

	for size, files := range sizeMap.sm {

		if len(files) > 1 {
			fmt.Println("Size : ", size)

			for _, fileItem := range files {
				fmt.Println(fileItem)
			}
		}

	}

}
