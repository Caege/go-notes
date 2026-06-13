package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Fetcher interface {
	Fetch(url string) (body string, urls []string, err error)
}

type Result struct {
	body string
	urls []string
	err  error
}

type SafeMap struct {
	mu sync.Mutex
	m  map[string]Result
}

func Crawl(url string, depth int, fetcher Fetcher, sm *SafeMap, wg *sync.WaitGroup) {

	defer wg.Done()
	if depth <= 0 {
		return
	}

	sm.mu.Lock()
	if _, ok := sm.m[url]; !ok {
		sm.m[url] = Result{}

		sm.mu.Unlock()
	} else {
		sm.mu.Unlock()

		return
	}

	body, urls, err := fetcher.Fetch(url)

	if err != nil {
		fmt.Println("Nothing found : ", url)
		return
	}

	sm.mu.Lock()

	(sm.m)[url] = Result{
		body, urls, nil,
	}

	sm.mu.Unlock()

	fmt.Printf("body : %s, url :%s \n", body, url)

	for _, v := range urls {

		wg.Add(1)
		go Crawl(v, depth-1, fetcher, sm, wg)
	}

	return
}

type FakeResult struct {
	body string
	urls []string
}

type FakeMap map[string]*FakeResult

func (f FakeMap) Fetch(url string) (body string, urls []string, err error) {
	if res, ok := f[url]; ok {
		time.Sleep(20 * time.Millisecond)
		return res.body, res.urls, nil
	}

	return "", nil, fmt.Errorf("error bruh : %s", url)
}

func main() {

	var wg sync.WaitGroup

	visitedSafeMap := SafeMap{m: map[string]Result{}}

	fakeMap := GenerateRandomWeb(10000, 5)

	wg.Add(1)
	go Crawl("page-0", 4, fakeMap, &visitedSafeMap, &wg)
	wg.Wait()
}

func GenerateRandomWeb(numPages int, linksPerPage int) FakeMap {
	fm := make(FakeMap)

	urls := make([]string, numPages)

	for i := 0; i < numPages; i++ {
		urls[i] = fmt.Sprintf("page-%d", i)
	}

	for _, url := range urls {
		var links []string

		for i := 0; i < linksPerPage; i++ {
			target := urls[rand.Intn(len(urls))]
			links = append(links, target)
		}

		fm[url] = &FakeResult{
			body: url,
			urls: links,
		}
	}

	return fm
}
