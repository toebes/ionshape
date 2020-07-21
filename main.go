package main

import (
	"context"
	"flag"
	"fmt"
	"os"

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

var (
	// Command-line flags
	folderIDs    arrayFlags
	apiAccessKey string
	apiSecretKey string
	onshapeDebug bool
	filepat      string
	dirpat       string
	fixvendor    string
)

func main() {
	flag.BoolVar(&onshapeDebug, "debug", false, "enable Onshape API debugging")
	flag.StringVar(&filepat, "name", "", "file pattern to match")
	flag.StringVar(&dirpat, "dir", "", "directory name pattern to match")
	flag.StringVar(&fixvendor, "fixvendor", "", "Vendor name to update parts and assemblies with")
	flag.StringVar(&apiSecretKey, "secret", "", "Onshape API Secret key")
	flag.StringVar(&apiAccessKey, "access", "", "Onshape API Access key")
	flag.Var(&folderIDs, "fid", "folder id(s) to include in scan")
	flag.Parse()

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

	processFolders(ctx, client)
}
