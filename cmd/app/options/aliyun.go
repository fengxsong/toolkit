package options

import (
	"errors"

	"github.com/spf13/pflag"
)

type AliyunCommonOption struct {
	AccessKey       string
	AccessKeySecret string
	Endpoint        string
	DryRun          bool
}

func (o *AliyunCommonOption) Validate() error {
	o.Endpoint = GetEnvWithDefault("ALIYUN_ENDPOINT", o.Endpoint)
	if o.Endpoint == "" {
		return errors.New(`required flag(s) "endpoint" not set`)
	}
	return nil
}

func (o *AliyunCommonOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.AccessKey, "ak", GetEnvWithDefault("ALIYUN_ACCESSKEY", ""), "accesskey id")
	fs.StringVar(&o.AccessKeySecret, "sk", GetEnvWithDefault("ALIYUN_ACCESSKEY_SECRET", ""), "accesskey secret")
	fs.StringVar(&o.Endpoint, "endpoint", "", "aliyun service endpoint")
	fs.BoolVar(&o.DryRun, "dry-run", false, "simulate execution without actually run")
}
