package url_processor

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type UrlProcessor struct {
	wg                      *sync.WaitGroup
	logger                  *log.Logger
	client                  *http.Client
	maxConcurrentUrlProcess int
	urlConnectionTimeout    time.Duration
}

func NewUrlProcessor(
	wg *sync.WaitGroup,
	logger *log.Logger,
	client *http.Client,
	maxConcurrentUrlProcess int,
	urlConnectionTimeout time.Duration,
) *UrlProcessor {
	return &UrlProcessor{
		wg:                      wg,
		logger:                  logger,
		client:                  client,
		maxConcurrentUrlProcess: maxConcurrentUrlProcess,
		urlConnectionTimeout:    urlConnectionTimeout,
	}
}

func (p *UrlProcessor) ProcessUrls(
	ctx context.Context,
	urls []string,
) (processedResult map[string]string, err error) {
	mx := &sync.Mutex{}
	processedResult = map[string]string{}

	wg := new(sync.WaitGroup)
	urlChannel := make(chan string, p.maxConcurrentUrlProcess)

	for w := 0; w < p.maxConcurrentUrlProcess; w++ {
		p.wg.Add(1)
		wg.Add(1)

		go func() {
			defer func() {
				p.wg.Done()
				wg.Done()
			}()

			for {
				select {
				case <-ctx.Done():
					err = ctx.Err()
					return
				case url, ok := <-urlChannel:
					if !ok {
						return
					}

					body, perr := p.processUrl(ctx, url)
					if perr != nil {
						err = perr
						return
					}

					mx.Lock()
					processedResult[url] = body
					mx.Unlock()
				}
			}
		}()
	}

	for _, url := range urls {
		urlChannel <- url
	}

	close(urlChannel)
	wg.Wait()

	if err != nil {
		return nil, err
	}

	return processedResult, nil
}

func (p *UrlProcessor) processUrl(ctx context.Context, url string) (string, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, p.urlConnectionTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctxTimeout, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
