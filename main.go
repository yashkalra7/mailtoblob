package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mailtoblob/blob"
	"mailtoblob/config"
	"mailtoblob/logger"
	"mailtoblob/router"
	"mailtoblob/sysexits"
)

// Conf hold the local configuration for mailtoblob
var Conf config.Config
var address string
var from string

func init() {
	// get the configuration file, set region, flags and mail routes
	Conf = config.Load()

	usageA := "email address for the receiving mailbox"
	flag.StringVar(&address, "address", "CatchAll", usageA)
	flag.StringVar(&address, "a", "CatchAll", usageA+" (shorthand)")

	usageF := "sender address, pass postfix ${sasl_sender} or ${sender}"
	flag.StringVar(&from, "from", "", usageF)
	flag.StringVar(&from, "f", "", usageF+" (shorthand)")
}

func main() {

	// retrieve the flags
	flag.Parse()

	objectKey := generateNameHash()

	logger.Log.Printf("[INFO] processing message from=%s, to=%s, object=%s", from, address, objectKey)

	// find matching mailbox
	// if matching mailbox found read body and pass to put object
	if m, ok := router.MatchMailbox(Conf.Mailboxes, address); ok {

		// retrieve message body passed as argument to mailtoblob
		msgBody, err := getBody()
		if err != nil {
			logger.Log.Printf("[ERROR] %s", err)
			// let mta know that there was I/O error
			os.Exit(sysexits.EX_NOINPUT)
		}

		prefix := formatPrefix(m.Prefix)
		azureConf := Conf.AzureConfig
		azureConf.ContainerName = m.ContainerName

		// Retry logic for blob upload
		for i := 0; i < 3; i++ {
			err = blob.UploadFileToAzureBlobStorage(&azureConf, &address, &msgBody, objectKey, prefix)
			if err == nil {
				break
			}
			logger.Log.Printf("[ERROR] Failed to upload to Azure Blob Storage: %s. Attempt %d/3", err, i+1)
			time.Sleep(time.Second * 2)
		}
		if err != nil {
			logger.Log.Printf("[ERROR] Failed to upload to Azure Blob Storage after 3 attempts: %s", err)
			os.Exit(sysexits.EX_UNAVAILABLE)
		}

	} else {
		logger.Log.Printf("[WARNING] mailbox not found for: %s", address)
		os.Exit(sysexits.EX_NOUSER)
	}
}

func getBody() (string, error) {

	// read from stdin in first if there is no data check args
	// check if there is anything on stdin
	info, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}

	// if we have something on stdin read until EOF
	if info.Mode()&os.ModeNamedPipe != 0 {

		bytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return "", errors.New("mailtoblob error reading from stdin")
		}
		return string(bytes), nil

	}

	// nothing passed from pipe check args instead
	args := flag.Args()
	if len(args) != 1 {
		return "", errors.New("mailtoblob expects message body to be passed as the last argument or from stdin")
	}
	return args[0], nil

}

func formatPrefix(prefix string) string {
	re, err := regexp.Compile(`^dateTimeFormat\(.+\)$`)
	if err != nil {
		logger.Log.Printf("[WARNING] Unable to compile regex, prefix will be dropped, %s", fmt.Sprint(err))
		return ""
	}
	if re.MatchString(prefix) {

		dateLayout := strings.TrimLeft(strings.TrimRight(prefix, ")"), "dateTimeFormat(")
		return time.Now().Format(dateLayout)
	}
	return prefix
}

func generateNameHash() string {
	// get the sha1 string from current unix time
	h := sha1.New()
	s := strconv.FormatInt(time.Now().UnixNano(), 10)
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
