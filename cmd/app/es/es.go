package es

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/fengxsong/toolkit/cmd/app/factory"
	"github.com/fengxsong/toolkit/pkg/log"
)

const name = "es"

func init() {
	factory.Register(name, newSubCommand())
}

type commonOptions struct {
	username      string
	password      string
	esURL         string
	kibanaURL     string
	kibanaVersion string
	dryRun        bool
	timeout       time.Duration
}

func (o *commonOptions) setDefaults() {
	if o.esURL != "" && !strings.HasPrefix(o.esURL, "http://") && !strings.HasPrefix(o.esURL, "https://") {
		o.esURL = "http://" + o.esURL
	}
	if o.kibanaURL != "" && !strings.HasPrefix(o.kibanaURL, "http://") && !strings.HasPrefix(o.kibanaURL, "https://") {
		o.kibanaURL = "http://" + o.kibanaURL
	}
}

func (o *commonOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.username, "username", "u", "", "Username for es basicauth")
	fs.StringVarP(&o.password, "password", "p", "", "Password for user")
	fs.StringVar(&o.esURL, "es-url", "", "Elasticsearch URL")
	fs.StringVar(&o.kibanaURL, "kibana-url", "", "Kibana URL")
	fs.StringVar(&o.kibanaVersion, "kibana-version", "7.14.2", "Kibana version for in HTTP request header")
	fs.BoolVar(&o.dryRun, "dry-run", false, "Simulate but not actually run")
	fs.DurationVar(&o.timeout, "timeout", 10*time.Second, "Http client timeout")
}

func (o *commonOptions) complete() (c *client, err error) {
	c = &client{
		httpClient: &http.Client{Timeout: o.timeout},
		username:   o.username,
		password:   o.password,
		kbnVer:     o.kibanaVersion,
		dryRun:     o.dryRun,
	}
	if len(o.esURL) > 0 {
		if c.esURL, err = url.Parse(o.esURL); err != nil {
			return nil, err
		}
	}
	if len(o.kibanaURL) > 0 {
		if c.kibanaURL, err = url.Parse(o.kibanaURL); err != nil {
			return nil, err
		}
	}
	if err = c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

type client struct {
	httpClient *http.Client
	esURL      *url.URL
	kibanaURL  *url.URL
	username   string
	password   string
	kbnVer     string
	dryRun     bool
}

func (c *client) validate() error {
	if c.esURL != nil && c.esURL.Host == "" {
		return errors.New("invalid es url")
	}
	if c.kibanaURL != nil && c.kibanaURL.Host == "" {
		return errors.New("invalid kibana url")
	}
	return nil
}

func (c *client) doRequest(method, url string, data io.Reader, dryRun bool) (body []byte, err error) {
	start := time.Now()
	defer func() {
		log.GetLogger().Debugw("do request",
			"method", method,
			"url", url,
			"dry-run", dryRun,
			"duration", time.Since(start).String(),
			"body", string(body),
		)
	}()
	req, err := http.NewRequest(method, url, data)
	if err != nil {
		return nil, err
	}
	if dryRun {
		return nil, nil
	}
	req.Header.Add("kbn-version", c.kbnVer)
	c.setBasicAuthIfRequired(req)
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	delay := 500 * time.Microsecond

	var resp *http.Response
	for delay <= defaultRetryMaxDelay {
		resp, err = c.httpClient.Do(req)
		if err == nil {
			break
		}
		log.GetLogger().Warnf("error occur: %v, retrying", err)
		time.Sleep(delay)
		delay *= 2
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 >= 4 {
		return nil, fmt.Errorf("unexpected error: %s", string(body))
	}
	return body, nil
}

func newSubCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: "elastic stack toolkit",
	}
	cmd.AddCommand(newCreatePatternCommand())
	cmd.AddCommand(newDeletePatternCommand())
	cmd.AddCommand(newBulkRequestCommand())
	return cmd
}
