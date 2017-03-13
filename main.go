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
	"strings"
	"time"
)

func main() {
	agent := flag.Bool("agent", false, "run as agent")
	flag.Parse()
	if !*agent {
		http.HandleFunc("/", printInfo)
		http.HandleFunc("/in", receiveInfo)
		log.Fatal(http.ListenAndServe(":7160", nil))
	}
	rand.Seed(time.Now().UnixNano())
	time.Sleep(time.Duration(rand.Intn(60)) * time.Second)
	for range time.Tick(60 * time.Second) {
		sendInfo()
	}
}
func sendInfo() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Send info: %v\n", r)
		}
	}()
	resp, err := http.Post("http://admin:7160/in", "text/plain", strings.NewReader(getInfo()))
	if err != nil {
		log.Printf("Post error: %v\n", err)
	}
	if err = resp.Body.Close(); err != nil {
		log.Printf("Close resp body: %v\n", err)
	}
}

const FORMATTER = "%-6s%6s%6s%10s%6s%6s"

func printInfo(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("time " + fmt.Sprintf(FORMATTER, "node", "%cpu", "%mem", "top-user", "%cpu", "%mem") + "\n"))
	for _, s := range names {
		w.Write([]byte(infoTime[s].Format("1504 ") + infoContent[s]))
		w.Write([]byte("\n"))
	}
}

var names = []string{ // "admin",
	"node1", "node2", "node3", "node4", "node5",
	"node6", "node7", "node8", "node9", "node10",
	"node11", "node12", "node13", "node14",
	"node15", "node16", "node17", "node18",
}
var infoTime = map[string]time.Time{}
var infoContent = map[string]string{}

func receiveInfo(w http.ResponseWriter, r *http.Request) {
	var info string
	if bs, err := ioutil.ReadAll(r.Body); err != nil {
		info = fmt.Sprintf("Read request error: %v", err)
	} else {
		info = string(bs)
	}
	name := SPLIT_BY_SPACE.Split(info, 2)[0]
	infoTime[name] = time.Now()
	infoContent[name] = info
	w.Write([]byte("OK"))
}

func getInfo() string {
	node := hostname()
	cpu := fmt.Sprintf("%.1f", getCPUUsage()*100)
	mem := fmt.Sprintf("%.1f", getMemUsage()*100)
	user, uCPU, uMem := getUserInfo()
	info := fmt.Sprintf(FORMATTER, node, cpu, mem, user, uCPU, uMem)
	return info
}

var SPLIT_BY_SPACE = regexp.MustCompile(`\s+`)

func getUserInfo() (string, string, string) {
	first := strings.Split(cmd("ps", "-aux", "--sort=-pcpu"), "\n")[1]
	firsts := SPLIT_BY_SPACE.Split(first, 5)
	user := firsts[0]
	uCPU := firsts[2]
	uMem := firsts[3]
	return user, uCPU, uMem
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
