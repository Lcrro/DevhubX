//go:build ignore

package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/Lcrro/DevhubX/internal/cover"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: go run ./scripts/capture-workbench.go URL PNG")
	}
	browser := cover.Browser()
	if browser == "" {
		log.Fatal("Chrome / Edge / Chromium not found")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(browser),
		chromedp.WindowSize(1440, 900),
		chromedp.Flag("disable-background-networking", true),
	)
	alloc, stop := chromedp.NewExecAllocator(ctx, opts...)
	defer stop()
	tab, closeTab := chromedp.NewContext(alloc)
	defer closeTab()
	var png []byte
	if err := chromedp.Run(tab,
		chromedp.EmulateViewport(1440, 900),
		chromedp.Navigate(os.Args[1]),
		chromedp.WaitVisible(`aside.sidebar`, chromedp.ByQuery),
		chromedp.Sleep(1500*time.Millisecond),
		chromedp.CaptureScreenshot(&png),
	); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(os.Args[2], png, 0644); err != nil {
		log.Fatal(err)
	}
}
