package fusionsolar

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

const (
	// browserOpAttempts bounds retries for one CDP call. The caller can still
	// reload the page and try again; this only covers a short blip.
	browserOpAttempts = 2
	// browserOpTimeout stops a single stuck CDP call before the whole run deadline.
	browserOpTimeout = 30 * time.Second
	browserOpBackoff = time.Second
)

// isRetryableBrowserErr reports whether a CDP or navigation error is worth
// repeating while the run context is still active.
// Connection refused is not retryable: the CDP endpoint is down.
func isRetryableBrowserErr(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "connection refused") {
		return false
	}
	return strings.Contains(msg, "eof") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "websocket") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "use of closed") ||
		strings.Contains(msg, "i/o timeout")
}

// retryBrowserOp runs fn with a per-call timeout. Transport failures are
// retried until attempts are exhausted. A canceled or expired run context
// stops immediately so a real job timeout is not hidden by more retries.
func retryBrowserOp(ctx context.Context, op string, attempts int, timeout, backoff time.Duration, fn func(context.Context) error) error {
	if attempts < 1 {
		attempts = 1
	}
	var last error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			if last != nil {
				return fmt.Errorf("%s: %v: %w", op, last, err)
			}
			return fmt.Errorf("%s: %w", op, err)
		}

		err := func() error {
			callCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return fn(callCtx)
		}()
		if err == nil {
			return nil
		}
		last = err
		if !isRetryableBrowserErr(err) || ctx.Err() != nil || attempt == attempts {
			if ctx.Err() != nil {
				return fmt.Errorf("%s: %v: %w", op, err, ctx.Err())
			}
			if attempt == attempts && isRetryableBrowserErr(err) {
				return fmt.Errorf("%s after %d attempts: %w", op, attempts, err)
			}
			return fmt.Errorf("%s: %w", op, err)
		}

		slog.Warn("[*] Operação do navegador falhou, tentando novamente",
			"op", op,
			"attempt", attempt,
			"maxAttempts", attempts,
			"error", err,
		)
		if sleepErr := sleepContext(ctx, time.Duration(attempt)*backoff); sleepErr != nil {
			return fmt.Errorf("%s: %v: %w", op, last, sleepErr)
		}
	}
	return fmt.Errorf("%s: %w", op, last)
}

func retryBrowser(ctx context.Context, op string, fn func(context.Context) error) error {
	return retryBrowserOp(ctx, op, browserOpAttempts, browserOpTimeout, browserOpBackoff, fn)
}

func bindElements(parent context.Context, found rod.Elements) rod.Elements {
	els := make(rod.Elements, len(found))
	for i, el := range found {
		els[i] = el.Context(parent)
	}
	return els
}

func pageElements(ctx context.Context, page *rod.Page, selector string) (rod.Elements, error) {
	var els rod.Elements
	err := retryBrowser(ctx, fmt.Sprintf("query elements %q", selector), func(callCtx context.Context) error {
		found, err := page.Context(callCtx).Elements(selector)
		if err != nil {
			return err
		}
		els = bindElements(page.GetContext(), found)
		return nil
	})
	return els, err
}

func elementElements(ctx context.Context, el *rod.Element, selector string) (rod.Elements, error) {
	var els rod.Elements
	// Keep the element's stable context. el.Page() may be a short-lived clone
	// from the query that found it, and that clone's context is already canceled.
	parent := el.GetContext()
	err := retryBrowser(ctx, fmt.Sprintf("query elements %q", selector), func(callCtx context.Context) error {
		found, err := el.Context(callCtx).Elements(selector)
		if err != nil {
			return err
		}
		els = bindElements(parent, found)
		return nil
	})
	return els, err
}

func waitElement(ctx context.Context, page *rod.Page, selector string) (*rod.Element, error) {
	var found *rod.Element
	err := retryBrowser(ctx, fmt.Sprintf("wait element %q", selector), func(callCtx context.Context) error {
		el, err := page.Context(callCtx).Element(selector)
		if err != nil {
			return err
		}
		found = el.Context(page.GetContext())
		return nil
	})
	return found, err
}

func elementText(ctx context.Context, el *rod.Element) (string, error) {
	var text string
	err := retryBrowser(ctx, "read element text", func(callCtx context.Context) error {
		var err error
		text, err = el.Context(callCtx).Text()
		return err
	})
	return text, err
}

func elementClick(ctx context.Context, el *rod.Element) error {
	return retryBrowser(ctx, "click element", func(callCtx context.Context) error {
		return el.Context(callCtx).Click(proto.InputMouseButtonLeft, 1)
	})
}

func elementProperty(ctx context.Context, el *rod.Element, name string) (string, error) {
	var value string
	err := retryBrowser(ctx, fmt.Sprintf("read element property %q", name), func(callCtx context.Context) error {
		prop, err := el.Context(callCtx).Property(name)
		if err != nil {
			return err
		}
		value = prop.String()
		return nil
	})
	return value, err
}

func inputText(ctx context.Context, el *rod.Element, text string) error {
	return retryBrowser(ctx, "input text", func(callCtx context.Context) error {
		return el.Context(callCtx).Input(text)
	})
}

func pressKey(ctx context.Context, el *rod.Element, key input.Key) error {
	return retryBrowser(ctx, "press key", func(callCtx context.Context) error {
		return el.Context(callCtx).Type(key)
	})
}

func navigatePage(ctx context.Context, page *rod.Page, url string) error {
	return retryBrowser(ctx, "navigate", func(callCtx context.Context) error {
		return page.Context(callCtx).Navigate(url)
	})
}

func waitPageLoad(ctx context.Context, page *rod.Page) error {
	return retryBrowser(ctx, "wait page load", func(callCtx context.Context) error {
		return page.Context(callCtx).WaitLoad()
	})
}

func pageURL(ctx context.Context, page *rod.Page) (string, error) {
	var url string
	err := retryBrowser(ctx, "read page url", func(callCtx context.Context) error {
		info, err := page.Context(callCtx).Info()
		if err != nil {
			return err
		}
		if info == nil {
			return errors.New("empty page info")
		}
		url = info.URL
		return nil
	})
	return url, err
}

func readCanvasDataURL(ctx context.Context, canvas *rod.Element) (string, error) {
	var dataURL string
	err := retryBrowser(ctx, "read canvas image", func(callCtx context.Context) error {
		res, err := canvas.Context(callCtx).Eval(`function() { return this.toDataURL('image/png') }`)
		if err != nil {
			return err
		}
		if res == nil {
			return errors.New("empty canvas result")
		}
		dataURL = res.Value.Str()
		return nil
	})
	return dataURL, err
}
