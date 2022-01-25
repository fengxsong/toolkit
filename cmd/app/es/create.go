package es

import (
	"bytes"
	"encoding/json"
	"fmt"
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

type createOptions struct {
	*commonOptions
	tsFieldName   string
	namespace     string
	filter        string
	skipDotPrefix bool
	override      bool
	refresh       bool
}

func (o *createOptions) setDefaults() {
	o.commonOptions.setDefaults()
	if o.tsFieldName == "" {
		o.tsFieldName = "@timestamp"
	}
}

func newCreatePatternCommand() *cobra.Command {
	o := &createOptions{
		commonOptions: &commonOptions{},
	}
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"register"},
		Short:   "Create index pattern[s] in kibana",
		RunE: func(_ *cobra.Command, _ []string) error {
			o.setDefaults()
			return o.Run()
		},
	}
	o.AddFlags(cmd.Flags())
	cmd.MarkFlagRequired("es-url")
	cmd.MarkFlagRequired("kibana-url")
	cmd.Flags().StringVar(&o.tsFieldName, "ts", "@timestamp", "Fieldname of timestamp")
	cmd.Flags().StringVarP(&o.namespace, "namespace", "n", "default", "Kibana namespace")
	cmd.Flags().StringVarP(&o.filter, "filter", "f", "", "Regexp pattern to filter, usually used to match prefix")
	cmd.Flags().BoolVar(&o.override, "override", false, "Overrides an existing index pattern if an index pattern with the provided title already exists")
	cmd.Flags().BoolVar(&o.refresh, "refresh", false, "Reloads index pattern fields after the index pattern is stored")
	cmd.Flags().BoolVar(&o.skipDotPrefix, "skip-dot-prefix", true, "Skip indices with `.` prefix")
	cmd.Flags().MarkHidden("skip-dot-prefix")
	return cmd
}

func (o *createOptions) Run() (err error) {
	cli, err := o.commonOptions.complete()
	if err != nil {
		return err
	}
	var filterPatternReg *regexp.Regexp
	if len(o.filter) > 0 {
		filterPatternReg = regexp.MustCompile(o.filter)
	}
	indicePatterns, err := cli.listIndices(o.skipDotPrefix, filterPatternReg)
	if err != nil {
		return fmt.Errorf("fetch indices: %s", err)
	}
	for i := range indicePatterns {
		if err = cli.createPattern(o.namespace, indicePatterns[i], o.tsFieldName, o.override, o.refresh); err != nil {
			return err
		}
		log.GetLogger().Infof("create index pattern for %s", indicePatterns[i])
	}
	return nil
}

func (c *client) setBasicAuthIfRequired(req *http.Request) {
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

func (c *client) listIndices(skipDotPrefix bool, filterPatternReg *regexp.Regexp) ([]string, error) {
	c.esURL.Path = "/_cat/indices"
	c.esURL.RawQuery = url.Values{"format": []string{"json"}}.Encode()
	if filterPatternReg != nil {
		pt := filterPatternReg.String()
		if strings.HasPrefix(pt, "^") {
			c.esURL.Path = path.Join(c.esURL.Path, strings.TrimPrefix(pt, "^"))
		}
	}
	respBody, err := c.doRequest(http.MethodGet, c.esURL.String(), nil, false)
	if err != nil {
		return nil, err
	}
	var indices []struct {
		Status string `json:"status"`
		Index  string `json:"index"`
	}
	if err = json.Unmarshal(respBody, &indices); err != nil {
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

func (c *client) createPattern(namespace, s, tsFieldName string, override, refresh bool) error {
	c.kibanaURL.Path = fmt.Sprintf("/s/%s/api/index_patterns/index_pattern", namespace)
	pl := map[string]interface{}{
		"override":       override,
		"refresh_fields": refresh,
		"index_pattern": map[string]string{
			"id":            s,
			"title":         fmt.Sprintf("%s-*", s),
			"timeFieldName": tsFieldName,
		},
	}
	b, err := json.Marshal(&pl)
	if err != nil {
		return err
	}
	_, err = c.doRequest(http.MethodPost, c.kibanaURL.String(), bytes.NewReader(b), c.dryRun)
	return err
}
