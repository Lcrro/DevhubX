package cover

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func LocalURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	h := u.Hostname()
	return h == "localhost" || (net.ParseIP(h) != nil && net.ParseIP(h).IsLoopback())
}
func Browser() string {
	if p := os.Getenv("DEVHUB_BROWSER"); p != "" {
		return p
	}
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser", "msedge", "chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	var paths []string
	switch runtime.GOOS {
	case "windows":
		for _, root := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LOCALAPPDATA")} {
			paths = append(paths, filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"), filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe"))
		}
	case "darwin":
		paths = []string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"}
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
func Capture(ctx context.Context, raw, path string) error {
	if !LocalURL(raw) {
		return errors.New("截图仅允许本机 HTTP/HTTPS 地址")
	}
	browser := Browser()
	if browser == "" {
		return errors.New("未找到 Chrome / Edge / Chromium；可通过 DEVHUB_BROWSER 指定")
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	options := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(browser), chromedp.WindowSize(1280, 800), chromedp.Flag("disable-background-networking", true), chromedp.Flag("no-sandbox", false))
	allocator, free := chromedp.NewExecAllocator(ctx, options...)
	defer free()
	tab, closeTab := chromedp.NewContext(allocator)
	defer closeTab()
	// Interception applies to redirects and subresources, not only the initial URL.
	chromedp.ListenTarget(tab, func(ev any) {
		if e, ok := ev.(*fetch.EventRequestPaused); ok {
			go func() {
				c := chromedp.FromContext(tab)
				if c == nil || c.Target == nil {
					return
				}
				executor := cdp.WithExecutor(tab, c.Target)
				u := e.Request.URL
				if LocalURL(u) {
					_ = fetch.ContinueRequest(e.RequestID).Do(executor)
				} else {
					_ = fetch.FailRequest(e.RequestID, network.ErrorReasonBlockedByClient).Do(executor)
				}
			}()
		}
	})
	var png []byte
	if err := chromedp.Run(tab, fetch.Enable(), chromedp.EmulateViewport(1280, 800), chromedp.Navigate(raw), chromedp.WaitReady("body", chromedp.ByQuery), chromedp.Sleep(800*time.Millisecond), chromedp.CaptureScreenshot(&png)); err != nil {
		return err
	}
	if err := os.WriteFile(path+".tmp", png, 0600); err != nil {
		return err
	}
	defer os.Remove(path + ".tmp")
	return os.Rename(path+".tmp", path)
}
