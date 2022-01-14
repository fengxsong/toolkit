package es

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fengxsong/toolkit/pkg/log"
)

const (
	defaultRetryMaxDelay = 10 * time.Second
)

type options struct {
	username      string
	password      string
	esURL         string
	kibanaURL     string
	kibanaVersion string
	tsFieldName   string
	namespace     string
	filter        string
	skipDotPrefix bool
	dryRun        bool
}

func (o *options) setDefaults() {
	if !strings.HasPrefix(o.esURL, "http://") && !strings.HasPrefix(o.esURL, "https://") {
		o.esURL = "http://" + o.esURL
	}
	if !strings.HasPrefix(o.kibanaURL, "http://") && !strings.HasPrefix(o.kibanaURL, "https://") {
		o.kibanaURL = "http://" + o.kibanaURL
	}
}

func newRegsiterPatternCommand() *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{
		Use:     "register",
		Aliases: []string{"rg"},
		Short:   "Automatically register indices patterns for kibana",
		RunE: func(_ *cobra.Command, _ []string) error {
			o.setDefaults()
			return runRegisterPatterns(o)
		},
	}
	cmd.Flags().StringVarP(&o.username, "username", "u", "", "Username for es basicauth")
	cmd.Flags().StringVarP(&o.password, "password", "p", "", "Password for user")
	cmd.Flags().StringVar(&o.esURL, "es-url", "", "Elasticsearch URL")
	cmd.Flags().StringVar(&o.kibanaURL, "kibana-url", "", "Kibana URL")
	cmd.Flags().StringVar(&o.kibanaVersion, "kibana-version", "7.14.2", "Kibana version for in HTTP request header")
	cmd.Flags().StringVar(&o.tsFieldName, "ts", "@timestamp", "Fieldname of timestamp")
	cmd.Flags().StringVarP(&o.namespace, "namespace", "n", "default", "Kibana namespace")
	cmd.Flags().StringVarP(&o.filter, "filter", "f", "", "Regexp pattern to filter, usually used to match prefix")
	cmd.Flags().BoolVar(&o.skipDotPrefix, "skip-dot-prefix", true, "Skip indices with `.` prefix")
	cmd.Flags().MarkHidden("skip-dot-prefix")
	cmd.Flags().BoolVar(&o.dryRun, "dry-run", false, "Simulate register but not actually run")
	return cmd
}

func runRegisterPatterns(o *options) (err error) {
	cli := &client{
		httpClient: &http.Client{},
		username:   o.username,
		password:   o.password,
	}
	if cli.esURL, err = url.Parse(o.esURL); err != nil {
		return
	}
	if cli.esURL.Host == "" {
		return errors.New("invalid es url")
	}

	if cli.kibanaURL, err = url.Parse(o.kibanaURL); err != nil {
		return
	}
	if cli.kibanaURL.Host == "" {
		return errors.New("invalid kibana url")
	}

	var filterPatternReg *regexp.Regexp
	if len(o.filter) > 0 {
		filterPatternReg = regexp.MustCompile(o.filter)
	}
	indicePatterns, err := cli.listIndicePatterns(o.skipDotPrefix, filterPatternReg)
	if err != nil {
		return fmt.Errorf("fetch indices: %s", err)
	}
	for i := range indicePatterns {
		if err = cli.registerPattern(o.namespace, indicePatterns[i], o.kibanaVersion, o.tsFieldName, o.dryRun); err != nil {
			return err
		}
		log.GetLogger().Infof("register index pattern for %s", indicePatterns[i])
	}
	return nil
}

type client struct {
	httpClient *http.Client
	esURL      *url.URL
	kibanaURL  *url.URL
	username   string
	password   string
}

func (c *client) setBasicAuthIfRequired(req *http.Request) {
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

func (c *client) listIndicePatterns(skipDotPrefix bool, filterPatternReg *regexp.Regexp) ([]string, error) {
	c.esURL.Path = "/_cat/indices"
	c.esURL.RawQuery = url.Values{"format": []string{"json"}}.Encode()
	if filterPatternReg != nil {
		pt := filterPatternReg.String()
		if strings.HasPrefix(pt, "^") {
			c.esURL.Path = path.Join(c.esURL.Path, strings.TrimPrefix(pt, "^"))
		}
	}

	req, err := http.NewRequest(http.MethodGet, c.esURL.String(), nil)
	if err != nil {
		return nil, err
	}
	c.setBasicAuthIfRequired(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 > 3 {
		return nil, fmt.Errorf("unexpected error: %s", string(body))
	}
	var indices []struct {
		Status string `json:"status"`
		Index  string `json:"index"`
	}
	if err = json.Unmarshal(body, &indices); err != nil {
		return nil, err
	}
	temp := make(map[string]struct{})
	parsePatternReg := regexp.MustCompile(`([a-zA-Z0-9_-]*)-\d{4}.\d{2}.\d{2}`)

	for _, indice := range indices {
		if indice.Status != "open" || (skipDotPrefix && strings.HasPrefix(indice.Index, ".")) {
			continue
		}
		if filterPatternReg != nil && !filterPatternReg.Match([]byte(indice.Index)) {
			log.GetLogger().Debugf("skip pattern %s", indice.Index)
			continue
		}
		ret := parsePatternReg.FindStringSubmatch(indice.Index)
		if len(ret) != 2 || len(ret[1]) == 0 {
			continue
		}
		temp[ret[1]] = struct{}{}
	}
	patterns := make([]string, 0, len(temp))
	for k := range temp {
		patterns = append(patterns, k)
	}
	return patterns, nil
}

func (c *client) registerPattern(namespace string, s string, kbnVer string, tsFieldName string, dryRun bool) error {
	if tsFieldName == "" {
		tsFieldName = "@timestamp"
	}
	c.kibanaURL.Path = fmt.Sprintf("/s/%s/api/saved_objects/index-pattern/%s", namespace, s)
	pl := map[string]interface{}{
		"attributes": map[string]string{
			"title":         fmt.Sprintf("%s-*", s),
			"timeFieldName": tsFieldName,
		},
	}
	b, err := json.Marshal(&pl)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.kibanaURL.String(), bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Add("kbn-version", kbnVer)
	c.setBasicAuthIfRequired(req)
	if dryRun {
		return nil
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
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode/100 >= 4 {
		return fmt.Errorf("unexpected error: %s", string(body))
	}
	return nil
}
