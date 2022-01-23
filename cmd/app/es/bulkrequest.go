package es

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/fengxsong/toolkit/internal/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type bulkRequestOptions struct {
	*commonOptions
	concurrency int
}

func newBulkRequestCommand() *cobra.Command {
	o := &bulkRequestOptions{
		commonOptions: &commonOptions{},
	}
	cmd := &cobra.Command{
		Use:   "bulk",
		Short: "Perform bulkrequests to elasticsearch",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			o.setDefaults()
			return runBulkRequests(o, args...)
		},
	}
	o.AddFlags(cmd.Flags())
	cmd.MarkFlagRequired("es-url")
	cmd.Flags().IntVarP(&o.concurrency, "concurrency", "c", runtime.NumCPU(), "Concurrency number")
	return cmd
}

type Request struct {
	Method  string `json:"method" yaml:"method"`
	URLPath string `json:"url" yaml:"url"`
	Body    string `json:"body" yaml:"body"`
}

func (r *Request) doWithClient(c *client) ([]byte, error) {
	uri := *c.esURL
	uri.Path = r.URLPath
	var data io.Reader
	if len(r.Body) > 0 {
		data = bytes.NewReader([]byte(r.Body))
	}
	return c.doRequest(r.Method, uri.String(), data, c.dryRun)
}

func parseFile(fn string) ([]*Request, error) {
	fileContent, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(fn)
	var requests []*Request
	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(fileContent, &requests)
	case ".json":
		err = json.Unmarshal(fileContent, &requests)
	case ".list":
		requests, err = parseListFile(fileContent)
	default:
		return nil, fmt.Errorf("unknown file extension: %s", ext)
	}
	return requests, err
}

func parseListFile(data []byte) ([]*Request, error) {
	splits := bytes.Split(data, []byte("---\n"))
	reg := regexp.MustCompile(`([a-zA-Z]+)\s+(\S+)`)
	var requests []*Request
	for i := range splits {
		req, err := parseRaw(strings.TrimRight(strings.TrimLeft(string(splits[i]), " "), " "), reg)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, nil
}

func parseRaw(data string, firstLineReg *regexp.Regexp) (*Request, error) {
	splits := strings.SplitN(data, "\n", 2)
	matches := firstLineReg.FindStringSubmatch(string(splits[0]))
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid format: %s", splits[0])
	}
	return &Request{
		Method:  matches[1],
		URLPath: matches[2],
		Body:    strings.TrimRight(strings.TrimLeft(string(splits[1]), " "), " "),
	}, nil
}

func runBulkRequests(o *bulkRequestOptions, files ...string) error {
	cli, err := o.commonOptions.complete()
	if err != nil {
		return err
	}
	errCh := make(chan error, o.concurrency)
	go func() {
		var wg sync.WaitGroup
		for i := range files {
			requests, err := parseFile(files[i])
			if err != nil {
				errCh <- err
			}
			for j := range requests {
				wg.Add(1)
				go func(r *Request) {
					defer wg.Done()
					_, err := r.doWithClient(cli)
					errCh <- err
				}(requests[j])
			}
		}
		wg.Wait()
		close(errCh)
	}()
	var errs []error
	for e := range errCh {
		if e != nil {
			errs = append(errs, e)
		}
	}
	if len(errs) > 0 {
		return errors.MultiError(errs)
	}
	return nil
}
