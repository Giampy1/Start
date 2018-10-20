package main

import (
	"flag"
	"fmt"
	"log"
	_ "net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/kjk/notionapi"
)

var (
	analyticsCode = "UA-194516-1"

	flgRedownloadNotion bool
	flgRedownloadPage   string
	flgDeploy           bool
	flgPreview          bool
	flgVerbose          bool
)

func parseCmdLineFlags() {
	flag.BoolVar(&flgVerbose, "verbose", false, "if true, verbose logging")
	flag.BoolVar(&flgDeploy, "deploy", false, "if true, build for deployment")
	flag.BoolVar(&flgPreview, "preview", false, "if true, runs caddy and opens a browser for preview")
	flag.BoolVar(&flgRedownloadNotion, "redownload-notion", false, "if true, re-downloads content from notion")
	flag.StringVar(&flgRedownloadPage, "redownload-page", "", "if given, redownloads content for one page")
	flag.Parse()
}

func logVerbose(format string, args ...interface{}) {
	if !flgVerbose {
		return
	}
	fmt.Printf(format, args...)
}

func rebuildAll(c *notionapi.Client) {
	regenMd()
	loadTemplates()
	articles := loadArticles(c)
	readRedirects(articles)
	netlifyBuild(articles)
}

// caddy -log stdout
func runCaddy() {
	cmd := exec.Command("caddy", "-log", "stdout")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func stopCaddy(cmd *exec.Cmd) {
	cmd.Process.Kill()
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func preview() {
	go func() {
		time.Sleep(time.Second * 1)
		openBrowser("http://localhost:8080")
	}()
	runCaddy()
}

func main() {
	parseCmdLineFlags()
	os.MkdirAll("netlify_static", 0755)

	client := &notionapi.Client{}
	authToken, ok := os.LookupEnv("NOTION_TOKEN")
	if !ok || authToken != "" {
		fmt.Printf("Must set NOTION_TOKEN env variable!\n")
	}
	client.AuthToken = authToken

	// make sure this happens first so that building for deployment is not
	// disrupted by the temporary testing code we might have below
	if flgDeploy {
		rebuildAll(client)
		return
	}

	if flgRedownloadPage != "" {
		notionRedownloadOne(client, flgRedownloadPage)
		os.Exit(0)
	}

	if false {
		testNotionToHTMLOnePage(client, "dfbefe6906a943d8b554699341e997b0")
		os.Exit(0)
	}

	if flgRedownloadNotion {
		notionRedownloadAll(client)
		os.Exit(0)
	}

	rebuildAll(client)
	if flgPreview {
		preview()
	}
}
