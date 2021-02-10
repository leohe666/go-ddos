package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

type Task struct {
	Name string
}

type Agent struct {
	Host string
	Port string
}

func work(wg *sync.WaitGroup, jobs chan Task, w int, logger *log.Logger, filePath string, line int, agentList []string) {
	defer wg.Done()
	for true {
		job, ok := <-jobs
		if !ok {
			break
		}
		agent := getRandAgent(filePath, line, agentList)
		logger.Printf("[Thread-%d]Request-%s use agent host:%s, port:%s", w, job.Name, agent.Host, agent.Port)
	}
}

func readAgent(agentList *[]string, line *int, filePath string, logger *log.Logger, start time.Time) {
	logger.Printf("Read agent start")
	agentfile, err := os.OpenFile(filePath, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("代理文件打开失败", err)
		os.Exit(5)
	}
	defer agentfile.Close()
	reader := bufio.NewReader(agentfile)
	for {
		str, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		*line++
		*agentList = append(*agentList, str)
	}
	logger.Printf("Read agent completed, cost: %s", time.Now().Sub(start))
}

func getRandAgent(filePath string, line int, agentList []string) Agent {
	max := line
	min := 1
	rand.Seed(time.Now().UnixNano())
	randLine := rand.Intn(max-min+1) + min
	agentAddr := strings.Split(agentList[randLine-1 : randLine][0], ":")
	agent := Agent{
		Host: "",
		Port: "",
	}
	if len(agentAddr) == 2 {
		agent = Agent{
			Host: agentAddr[0],
			Port: agentAddr[1],
		}
		return agent
	}
	return agent
}

func main() {
	start := time.Now()
	file, err := os.OpenFile("info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	gLogger := log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)
	gLogger.Println("Processing start")
	wg := &sync.WaitGroup{}
	jobs := make(chan Task)
	workers := 500
	runtime := 1
	filePath := ""
	flag.StringVar(&filePath, "f", "", "代理文件目录,支持相对路径,代理格式为 ip:port 每个代理一行")
	flag.IntVar(&workers, "w", 500, "线程数,默认500")
	flag.IntVar(&runtime, "s", 1, "运行时间,默认1秒,以秒为单位")
	flag.Parse()
	var agentList []string
	line := 0
	if filePath != "" {
		readAgent(&agentList, &line, filePath, gLogger, start)
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go work(wg, jobs, w, gLogger, filePath, line, agentList)
	}
	timeout := time.After(time.Second * time.Duration(runtime))
	finish := make(chan bool)
	count := 1
	go func() {
		for {
			select {
			case <-timeout:
				fmt.Println("Complete", runtime, "seconds data processing")
				finish <- true
				return
			default:
				task := Task{
					Name: fmt.Sprintf("请求-%d", count),
				}
				jobs <- task
				count++
			}
		}
	}()

	<-finish

	close(jobs)
	wg.Wait()
	gLogger.Printf("App processing completed, cost: %s", time.Now().Sub(start))
	fmt.Println("Processing completed")
}
