package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "time/tzdata"

	"github.com/steipete/sweetcookie"
	"golang.org/x/net/publicsuffix"
)

const portalURL = "https://uslugi.tatarstan.ru"

func browserSession(ctx context.Context, profile string, out io.Writer) (*http.Client, error) {
	options := sweetcookie.Options{URL: portalURL + "/gas/", Origins: []string{portalURL + "/personal-account/", portalURL + "/personal_account/"}, Browsers: []sweetcookie.Browser{sweetcookie.BrowserChrome}, Mode: sweetcookie.ModeFirst, Timeout: 15 * time.Second}
	if profile != "" {
		options.Profiles = map[sweetcookie.Browser]string{sweetcookie.BrowserChrome: profile}
	}
	result, err := sweetcookie.Get(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("SweetCookie could not read Chrome session: %w", err)
	}
	if len(result.Cookies) == 0 {
		return nil, fmt.Errorf("SweetCookie found no usable uslugi.tatarstan.ru cookies in Chrome; log in there first or select --profile (browser reported %d warnings)", len(result.Warnings))
	}
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("create session jar: %w", err)
	}
	origin, _ := url.Parse(portalURL)
	for _, cookie := range result.Cookies {
		value := &http.Cookie{Name: cookie.Name, Value: cookie.Value, Domain: cookie.Domain, Path: cookie.Path, Secure: cookie.Secure, HttpOnly: cookie.HTTPOnly}
		if cookie.Expires != nil {
			value.Expires = *cookie.Expires
		}
		jar.SetCookies(origin, []*http.Cookie{value})
	}
	fmt.Fprintf(out, "Authentication: SweetCookie imported %d portal cookies from Chrome.\n", len(result.Cookies))
	return &http.Client{Jar: jar, Timeout: 30 * time.Second}, nil
}

func run(ctx context.Context, args []string, out, errOut io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(errOut, "Usage: uslugi-tatar-cli {gas|main} [--dry-run] [--profile Chrome-profile] meters.yaml")
		if len(args) > 0 {
			return nil
		}
		return fmt.Errorf("expected the gas or main command")
	}
	if args[0] != "gas" && args[0] != "main" {
		return fmt.Errorf("unknown command %q; expected gas or main", args[0])
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(errOut)
	dryRun := flags.Bool("dry-run", false, "show account, meters and usage without submitting readings")
	profile := flags.String("profile", "", "Chrome profile name or directory (default: SweetCookie discovery)")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("%s requires one meters.yaml path; put flags before the path", args[0])
	}
	zone, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return fmt.Errorf("load Moscow timezone: %w", err)
	}
	now := time.Now().In(zone)
	var gas gasReading
	var main mainReading
	if args[0] == "gas" {
		gas, err = loadGasReading(flags.Arg(0), now)
	} else {
		main, err = loadMainReading(flags.Arg(0), now)
	}
	if err != nil {
		return err
	}
	httpClient, err := browserSession(ctx, *profile, out)
	if err != nil {
		return err
	}
	client, err := newPortalClient(portalURL, httpClient)
	if err != nil {
		return err
	}
	if args[0] == "gas" {
		return client.gas(ctx, gas, *dryRun, out)
	}
	return client.submitMain(ctx, main, *dryRun, out)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "uslugi-tatar-cli:", err)
		os.Exit(1)
	}
}
