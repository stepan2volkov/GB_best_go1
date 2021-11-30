package app

import (
	"context"
	"encoding/csv"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

//Crawler - интерфейс (контракт) краулера
type Crawler interface {
	Scan(ctx context.Context, url string, depth uint64)
	ChanResult() <-chan CrawlResult
	IncreaseMaxDepth(delta uint64)
}

//Config - структура для конфигурации
type Config struct {
	MaxDepth   uint64 `yaml:"max_depth"`
	MaxResults int    `yaml:"max_results"`
	MaxErrors  int    `yaml:"max_errors"`
	Url        string `yaml:"url"`
	Timeout    int    `yaml:"timeout"` //in seconds
	Delta      uint64 `yaml:"delta"`
	Output     string `yaml:"output"`
}

func Handle() {
	configPath := flag.String("config", "./configs/config.yaml", "the config path")
	flag.Parse()

	cfg := Config{}

	configFile, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf(`error when opening config in "%s": %v`, *configPath, err)
	}

	err = yaml.Unmarshal(configFile, &cfg)
	if err != nil {
		log.Fatalf("error when reading config: %v", err)
	}

	var r Requester = NewRequester(time.Duration(cfg.Timeout) * time.Second)
	var cr Crawler = NewCrawler(r, cfg.MaxDepth)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	go cr.Scan(ctx, cfg.Url, 0)            //Запускаем краулер в отдельной рутине
	go processResult(ctx, cancel, cr, cfg) //Обрабатываем результаты в отдельной рутине

	sigStopCh := make(chan os.Signal, 1)     //Создаем канал для приема сигнала завершения
	signal.Notify(sigStopCh, syscall.SIGINT) //Подписываемся на сигнал SIGINT

	sigCh := make(chan os.Signal, 1)      //Создаем канал для приема сигнала завершения
	signal.Notify(sigCh, syscall.SIGUSR1) //Подписываемся на сигнал SIGINT

	for {
		select {
		case <-ctx.Done(): //Если всё завершили - выходим
			return
		case <-sigStopCh:
			log.Println("Gracefull shutdown started") // Приятно знать, что мы действительно обработали сигнал
			cancel()                                  //Если пришёл сигнал SigInt - завершаем контекст
		case <-sigCh:
			cr.IncreaseMaxDepth(cfg.Delta)
			log.Printf("Max depth increased")
		}
	}
}

func processResult(ctx context.Context, cancel func(), cr Crawler, cfg Config) {
	var maxResult, maxErrors = cfg.MaxResults, cfg.MaxErrors

	outputFile := os.Stdout
	var err error

	if cfg.Output != "" {
		outputFile, err = os.Create(cfg.Output)
		if err != nil {
			log.Fatalf("error when openning csv-file: %v\n", err)
		}
	}
	defer outputFile.Close()
	csvWriter := csv.NewWriter(outputFile)

	// Формируем заголовок
	err = csvWriter.Write([]string{"URL", "Title"})
	if err != nil {
		log.Printf("error when writing csv header: %v\n", err)
		return
	}
	defer csvWriter.Flush()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-cr.ChanResult():
			if msg.Err != nil {
				maxErrors--
				log.Printf("crawler result return err: %s\n", msg.Err.Error())
				if maxErrors <= 0 {
					cancel()
					log.Println("Stopped because of max error limit exceeded")
					return
				}
			} else {
				maxResult--
				err := csvWriter.Write([]string{msg.Url, msg.Title})
				if err != nil {
					log.Println("Stopped because of error when ")
					return
				}

				// log.Printf("crawler result: [url: %s] Title: %s\n", msg.Url, msg.Title)
				if maxResult <= 0 {
					cancel()
					return
				}
			}
		}
	}
}
