package aliyun

import (
	"io"

	"github.com/fengxsong/toolkit/cmd/app/options"
	"github.com/spf13/cobra"
)

func newOssCommand(o *options.AliyunCommonOption, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oss",
		Short: "for aliyun oss service",
	}

	return cmd
}
