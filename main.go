package main

import (
	"flag"
	"fmt"
	"github.com/c9s/goprocinfo/linux"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	send := flag.String("send", "", "masters, separated by ','")
	listen := flag.String("listen", "", "port to listen")
	duration := flag.Int("duration", 60, "duration seconds")
	flag.Parse()
	if *listen != "" {
		http.HandleFunc("/", printInfo)
		http.HandleFunc("/in", receiveInfo)
		http.HandleFunc("/ver", printVer)
		log.Fatal(http.ListenAndServe(*listen, nil))
	}
	if *send != "" {
		urls := strings.Split(*send, ",")
		rand.Seed(time.Now().UnixNano())
		time.Sleep(time.Duration(rand.Intn(*duration)) * time.Second)
		for range time.Tick(time.Duration(*duration) * time.Second) {
			info := getInfo()
			for _, url := range urls {
				sendInfo(url, info)
			}
		}
	}
}

func sendInfo(url string, info string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Send info url: %s, error: %v\n", url, r)
		}
	}()
	resp, err := http.Post(url+"/in", "text/plain", strings.NewReader(info))
	if err != nil {
		log.Printf("Post error: %v\n", err)
		return
	}
	if err = resp.Body.Close(); err != nil {
		log.Printf("Close resp body error: %v\n", err)
	}
}

const FORMATTER = "%-7s%6s%5s%5s%8s%5s%5s"
const USER_NAME_LENGTH = 7
const NODE_NAME_LENGTH = 6

func printInfo(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("time " + fmt.Sprintf(FORMATTER, "node", "gpu", "%cpu", "%mem", "topuser", "%cpu", "%mem") + "\n"))
	names := []string{}
	infoMux.RLock()
	defer infoMux.RUnlock()
	for name := range infoContent {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := names[i], names[j]
		if len(a) != len(b) {
			return len(a) < len(b)
		}
		return a < b
	})
	for _, s := range names {
		w.Write([]byte(infoTime[s].Format("1504 ") + infoContent[s]))
		w.Write([]byte("\n"))
	}
	view++
}

var view uint64 = 0

func printVer(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("view: " + strconv.FormatUint(view, 10) + "\n"))
	w.Write([]byte("version: 2.0\n"))
	w.Write([]byte("by meijun\n"))
}

var infoTime = map[string]time.Time{}
var infoContent = map[string]string{}
var infoMux = sync.RWMutex{}

func receiveInfo(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Receive info: %v\n", r)
		}
	}()
	var info string
	if bs, err := ioutil.ReadAll(r.Body); err != nil {
		info = fmt.Sprintf("Read request error: %v", err)
	} else {
		info = string(bs)
	}
	name := SPLIT_BY_SPACE.Split(info, 2)[0]
	infoMux.Lock()
	defer infoMux.Unlock()
	infoTime[name] = time.Now()
	infoContent[name] = info
	w.Write([]byte("OK"))
}

func getInfo() string {
	node := hostname()
	cpu := fmt.Sprintf("%.1f", getCPUUsage()*100)
	mem := fmt.Sprintf("%.1f", getMemUsage()*100)
	gpuUsed, gpuAll := getGPUUsage()
	gpu := fmt.Sprintf("%d/%d", gpuUsed, gpuAll)
	cUser, uCPU := getUserCPU()
	mUser, uMem := getUserMem()
	if len(node) > NODE_NAME_LENGTH {
		node = node[len(node)-NODE_NAME_LENGTH:]
	}
	if len(cUser) > USER_NAME_LENGTH {
		cUser = cUser[len(cUser)-USER_NAME_LENGTH:]
	}
	if len(mUser) > USER_NAME_LENGTH {
		mUser = mUser[len(mUser)-USER_NAME_LENGTH:]
	}
	info := fmt.Sprintf(FORMATTER, node, gpu, cpu, mem, cUser, uCPU, uMem)
	return info
}

var SPLIT_BY_SPACE = regexp.MustCompile(`\s+`)

func getUserCPU() (string, string) {
	first := strings.Split(cmd("ps", "-aux", "--sort=-pcpu"), "\n")[1]
	firsts := SPLIT_BY_SPACE.Split(first, 5)
	user := firsts[0]
	uCPU := firsts[2]
	return user, uCPU
}
func getUserMem() (string, string) {
	first := strings.Split(cmd("ps", "-aux", "--sort=-pmem"), "\n")[1]
	firsts := SPLIT_BY_SPACE.Split(first, 5)
	user := firsts[0]
	uMem := firsts[3]
	return user, uMem
}

const GPU_MEMORY_THRESHOLD = 64

func getGPUUsage() (int, int) {
	output := cmd("nvidia-smi", "--query-gpu=memory.used", "--format=csv,noheader,nounits")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	used := 0
	all := 0
	for _, line := range lines {
		mem, err := strconv.ParseInt(line, 10, 0)
		if err != nil {
			return 0, 0
		}
		if mem > GPU_MEMORY_THRESHOLD {
			used++
		}
		all++
	}
	return used, all
}

func hostname() string {
	if name, err := os.Hostname(); err != nil {
		return fmt.Sprintf("Hostname error: %v", err)
	} else {
		return name
	}
}

var idle uint64
var total uint64

func getMemUsage() float64 {
	if mem, err := linux.ReadMemInfo("/proc/meminfo"); err != nil {
		return math.NaN()
	} else {
		return 1 - float64(mem.MemFree)/float64(mem.MemTotal)
	}
}

func getCPUUsage() float64 {
	stat, err := linux.ReadStat("/proc/stat")
	if err != nil {
		log.Printf("Read CPU info: %v\n", err)
		return math.NaN()
	}
	// http://stackoverflow.com/questions/9229333/how-to-get-overall-cpu-usage-e-g-57-on-linux
	all := stat.CPUStatAll
	idle2 := all.Idle + all.IOWait
	total2 := idle2 + all.User + all.Nice + all.System + all.IRQ + all.SoftIRQ + all.Steal
	idleDiff := idle2 - idle
	totalDiff := total2 - total
	idle = idle2
	total = total2
	return float64(totalDiff-idleDiff) / float64(totalDiff)
}

func cmd(name string, arg ...string) string {
	cmd := exec.Command(name, arg...)
	if bs, err := cmd.CombinedOutput(); err != nil {
		return fmt.Sprintf("cmd.CombinedOutput() output: %s error: %v", string(bs), err)
	} else {
		return string(bs)
	}
}
