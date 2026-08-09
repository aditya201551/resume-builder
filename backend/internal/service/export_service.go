package service

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

const (
	pdfPageWidthInches  = 680.0 / 96.0
	pdfPageHeightInches = 880.0 / 96.0
)

type ExportService struct {
	chromeExecPath string
}

func NewExportService(chromeExecPath string) *ExportService {
	return &ExportService{chromeExecPath: chromeExecPath}
}

func (s *ExportService) RenderPDF(ctx context.Context, printURL string) ([]byte, error) {
	ctx, cancelTimeout := context.WithTimeout(ctx, 20*time.Second)
	defer cancelTimeout()

	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.Flag("disable-gpu", true),
	)
	if s.chromeExecPath != "" {
		allocOpts = append(allocOpts, chromedp.ExecPath(s.chromeExecPath))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, allocOpts...)
	defer cancelAlloc()

	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	defer cancelBrowser()

	var pdfBuf []byte
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(printURL),
		chromedp.WaitVisible(`[data-print-ready]`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(pdfPageWidthInches).
				WithPaperHeight(pdfPageHeightInches).
				WithMarginTop(0).
				WithMarginBottom(0).
				WithMarginLeft(0).
				WithMarginRight(0).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuf = buf
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("render pdf: %w", err)
	}
	return pdfBuf, nil
}
