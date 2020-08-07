package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/toebes/go-client/onshape"
)

// arrayFlags generates a list of strings passed for command line parameters
type arrayFlags []string

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type workItem struct {
	order      int
	parentPath string
	element    onshape.BTGlobalTreeMagicNodeInfo
	finished   bool
}
type doneItem struct {
	order    int
	workerID int
	result   fileInfo
	err      error
	finished bool
}

var (
	// Command-line flags
	folderIDs    arrayFlags
	apiAccessKey string
	apiSecretKey string
	onshapeDebug bool
	filepat      string
	dirpat       string
	fixvendor    string
	logfile      string
	numWorkers   int
)

// MaxParallelism determines the maximum number of threads that it is reasonable to run
// From: https://stackoverflow.com/questions/13234749/golang-how-to-verify-number-of-processors-on-which-a-go-program-is-running
func MaxParallelism() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
}

func main() {
	flag.BoolVar(&onshapeDebug, "debug", false, "enable Onshape API debugging")
	flag.StringVar(&filepat, "name", "", "file pattern to match")
	flag.StringVar(&dirpat, "dir", "", "directory name pattern to match")
	flag.StringVar(&fixvendor, "fixvendor", "", "Vendor name to update parts and assemblies with")
	flag.StringVar(&apiSecretKey, "secret", "", "Onshape API Secret key")
	flag.StringVar(&apiAccessKey, "access", "", "Onshape API Access key")
	flag.StringVar(&logfile, "logfile", "outofshape.txt", "Log file to write generated names to")
	flag.IntVar(&numWorkers, "threads", MaxParallelism()-2, "Maximum number of worker threads")
	flag.Var(&folderIDs, "fid", "folder id(s) to include in scan")
	flag.Parse()

	// Queue globals
	workQueue := make(chan workItem, numWorkers*10)
	doneQueue := make(chan doneItem, numWorkers*10)
	allDone := make(chan bool, 1)

	config := onshape.NewConfiguration()
	config.Debug = onshapeDebug

	client := onshape.NewAPIClient(config)
	if apiSecretKey == "" {
		apiSecretKey = os.Getenv("ONSHAPE_API_SECRET_KEY")
	}
	if apiAccessKey == "" {
		apiAccessKey = os.Getenv("ONSHAPE_API_ACCESS_KEY")
	}
	if onshapeDebug {
		fmt.Printf("Keys: Secret=%v Access=%v\n", apiSecretKey, apiAccessKey)
	}

	ctx := context.WithValue(context.Background(), onshape.ContextAPIKeys, onshape.APIKeys{SecretKey: apiSecretKey, AccessKey: apiAccessKey})

	go outputThread(numWorkers, logfile, doneQueue, allDone)
	for i := 0; i < numWorkers; i++ {
		go fileThread(ctx, client, i, workQueue, doneQueue)
	}

	processed, err := processFolders(ctx, client, workQueue, doneQueue)
	if err != nil {
		fmt.Printf("***Folder Processing error: %v\n", err)
	}

	for i := 0; i < numWorkers; i++ {
		workQueue <- workItem{order: processed + 1, parentPath: "done", finished: true}
	}

	<-allDone

	fmt.Printf("All done")
}
