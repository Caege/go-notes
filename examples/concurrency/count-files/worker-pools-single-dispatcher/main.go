package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
)

type Results struct {
	newDirs []string
}

func main() {
	jobs := make(chan string)
	results := make(chan Results)
	var fileCounter atomic.Int64
	active := 0
	var pending []string

	pending = append(pending, `C:\`)

	workerCount := 4

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

					if entry.Type().IsRegular() {
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

	for {
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
			return

		}
	}

}
