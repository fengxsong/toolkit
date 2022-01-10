package aliyun

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/fengxsong/toolkit/cmd/app/options"
	"github.com/fengxsong/toolkit/pkg/log"
)

// todo: refactor functions

func newSlsCommand(o *options.AliyunCommonOption, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sls",
		Short: "for aliyun log service",
	}
	cmd.AddCommand(newProjectCommand(o, out))
	cmd.AddCommand(newLogstoreCommand(o, out))
	cmd.AddCommand(newShipperCommand(o))

	return cmd
}

func newProjectCommand(o *options.AliyunCommonOption, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "sls project",
	}
	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "list projects",
		Aliases: []string{"ls"},
		RunE: func(_ *cobra.Command, _ []string) error {
			client := sls.CreateNormalInterface(o.Endpoint, o.AccessKey, o.AccessKeySecret, "")
			projects, _, _, err := client.ListProjectV2(0, 500)
			if err != nil {
				return err
			}
			tw := tabwriter.NewWriter(out, 0, 0, 1, ' ', tabwriter.TabIndent)
			for _, p := range projects {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", p.Name, p.Description, p.Status)
			}
			return tw.Flush()
		},
	}
	{
		// todo: add ignore prefix support
		var (
			createIndex bool
			force       bool
		)
		syncCmd := &cobra.Command{
			Use:   "sync",
			Short: "sync logstore config from one project to another",
			Args:  cobra.MinimumNArgs(2),
			RunE: func(_ *cobra.Command, args []string) error {
				client := sls.CreateNormalInterface(o.Endpoint, o.AccessKey, o.AccessKeySecret, "")
				logstores, err := client.ListLogStore(args[0])
				if err != nil {
					return err
				}
				visitFunc := func(s string) error {
					ls, err := client.GetLogStore(args[0], s)
					if err != nil {
						return err
					}
					var index *sls.Index
					if createIndex {
						index, err = client.GetIndex(args[0], ls.Name)
						if err != nil {
							if slsErr, ok := err.(*sls.Error); !ok || slsErr.Code != "IndexConfigNotExist" {
								return err
							}
						}
					}
					return createLogstore(client, o.DryRun, args[1], ls.Name, ls.TTL, ls.ShardCount, ls.AutoSplit, ls.MaxSplitShard, force, index)
				}
				return visitAll(logstores, visitFunc)
			},
		}
		syncCmd.Flags().BoolVar(&createIndex, "create-index", false, "create index at the same time")
		syncCmd.Flags().BoolVar(&force, "force-update", false, "force update if already exists")

		cmd.AddCommand(syncCmd)
	}
	cmd.AddCommand(listCmd)
	return cmd
}

func newLogstoreCommand(o *options.AliyunCommonOption, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logstore",
		Short: "sls logstore",
	}
	var projectName string
	// list logstore sub command
	{
		listCmd := &cobra.Command{
			Use:     "list",
			Short:   "list logstore",
			Aliases: []string{"ls"},
			RunE: func(_ *cobra.Command, _ []string) error {
				client := sls.CreateNormalInterface(o.Endpoint, o.AccessKey, o.AccessKeySecret, "")
				tw := tabwriter.NewWriter(out, 0, 0, 1, ' ', tabwriter.TabIndent)
				logstores, err := client.ListLogStore(projectName)
				if err != nil {
					return err
				}
				fmt.Fprintf(tw, "%s\t%s\n", projectName, strings.Join(logstores, ","))
				return tw.Flush()
			},
		}
		cmd.AddCommand(listCmd)
	}
	// create logstore sub command
	{
		var (
			ttl, shardCnt, maxSplitShard  int
			createIndex, autoSplit, force bool
		)
		createCmd := &cobra.Command{
			Use:   "create",
			Short: "create logstore",
			Args:  cobra.MinimumNArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				client := sls.CreateNormalInterface(o.Endpoint, o.AccessKey, o.AccessKeySecret, "")

				visitFunc := func(s string) error {
					var index *sls.Index
					if createIndex {
						index = sls.CreateDefaultIndex()
						index.Line.Chn = true
					}
					return createLogstore(client, o.DryRun, projectName, s, ttl, shardCnt, autoSplit, maxSplitShard, force, index)
				}
				return visitAll(extractNames(args), visitFunc)
			},
		}
		createCmd.Flags().BoolVar(&createIndex, "create-index", false, "create index at the same time")
		createCmd.Flags().IntVar(&ttl, "ttl", 30, "time to live of logstore")
		createCmd.Flags().IntVar(&shardCnt, "shards", 2, "number of shards of logstore")
		createCmd.Flags().IntVar(&maxSplitShard, "max-split-shards", 32, "max number of shards of logstore")
		createCmd.Flags().BoolVar(&autoSplit, "auto-split", true, "auto split logstore")
		createCmd.Flags().BoolVar(&force, "force-update", false, "force update if already exists")
		cmd.AddCommand(createCmd)
	}
	{
		// delete logstore subcommand
		deleteCmd := &cobra.Command{
			Use:     "delete",
			Short:   "for remove named logstore",
			Aliases: []string{"remove"},
			Args:    cobra.MinimumNArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				client := sls.CreateNormalInterface(o.Endpoint, o.AccessKey, o.AccessKeySecret, "")
				project, err := client.GetProject(projectName)
				if err != nil {
					return err
				}
				return visitAll(extractNames(args), func(s string) (err error) {
					defer func() {
						if err == nil {
							log.GetLogger().Infof("delete logstore %s success", s)
						}
					}()
					if !o.DryRun {
						err = project.DeleteLogStore(s)
					}
					return
				})
			},
		}
		cmd.AddCommand(deleteCmd)
	}

	cmd.PersistentFlags().StringVarP(&projectName, "project", "p", "", "name of project")
	cmd.MarkPersistentFlagRequired("project")
	return cmd
}

type createShipperOption struct {
	ossBucket      string
	roleArn        string
	compressType   string
	pathFormat     string
	format         string
	bufferInterval int
	bufferSize     int
	force          bool
}

func (o *createShipperOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ossBucket, "oss-bucket", "", "name of oss bucket")
	fs.StringVar(&o.roleArn, "role-arn", "", "role arn")
	fs.StringVar(&o.compressType, "compress-type", "none", "compress type")
	fs.StringVar(&o.pathFormat, "path-format", `%Y/%m/%d/%H/%M`, "path format")
	fs.StringVar(&o.format, "format", `json`, "doc format")
	fs.IntVar(&o.bufferInterval, "buffer-interval", 300, "buffer interval")
	fs.IntVar(&o.bufferSize, "buffer-size", 256, "buffer size")
	fs.BoolVar(&o.force, "force-update", false, "force update if already exists")
}

func (o *createShipperOption) run(project *sls.LogProject, name string, dryRun bool) error {
	logstore, err := project.GetLogStore(name)
	if err != nil {
		return err
	}
	sc, err := logstore.GetShipper(name + "_to_oss")
	if err != nil {
		if slsErr, ok := err.(*sls.Error); !ok || !strings.Contains(slsErr.Message, "does not exist") {
			return err
		}
	}
	if !dryRun {
		ossSc := &sls.OSSShipperConfig{
			OssBucket:      o.ossBucket,
			OssPrefix:      name,
			RoleArn:        o.roleArn,
			BufferInterval: o.bufferInterval,
			BufferSize:     o.bufferSize,
			CompressType:   o.compressType,
			PathFormat:     o.pathFormat,
			// Format:         o.format,
			Storage: sls.ShipperStorage{
				Detail: map[string]interface{}{"columns": []string{}},
				Format: o.format,
			},
		}
		shipper := &sls.Shipper{
			ShipperName:         name + "_to_oss",
			TargetType:          sls.OSSShipperType,
			TargetConfiguration: ossSc,
		}
		if sc == nil {
			log.GetLogger().Debugf("creating logshipper for logstore %s", name)
			if err = logstore.CreateShipper(shipper); err != nil {
				return err
			}
		} else if o.force {
			log.GetLogger().Debugf("force updating logshipper for logstore %s", name)
			if err = logstore.UpdateShipper(shipper); err != nil {
				return err
			}
		}
	}
	log.GetLogger().Infof("create logshipper for %s success", name)
	return nil
}

func newShipperCommand(o *options.AliyunCommonOption) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shipper",
		Short: "for log shipper",
	}
	var projectName string
	{
		cso := &createShipperOption{}
		createCmd := &cobra.Command{
			Use:   "create",
			Short: "create log shipper to oss(currently supported), shipper name is same as name of each logstore",
			Args:  cobra.MinimumNArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				client := sls.CreateNormalInterface(o.Endpoint, o.AccessKey, o.AccessKeySecret, "")
				project, err := client.GetProject(projectName)
				if err != nil {
					return err
				}
				return visitAll(extractNames(args), func(s string) error { return cso.run(project, s, o.DryRun) })
			},
		}
		cso.AddFlags(createCmd.Flags())
		for _, fn := range []string{"project", "logstore", "oss-bucket", "role-arn"} {
			createCmd.MarkFlagRequired(fn)
		}
		cmd.AddCommand(createCmd)
	}
	{
		deleteCmd := &cobra.Command{
			Use:     "delete",
			Short:   "delete log shipper",
			Aliases: []string{"remove"},
			Args:    cobra.MinimumNArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				client := sls.CreateNormalInterface(o.Endpoint, o.AccessKey, o.AccessKeySecret, "")
				project, err := client.GetProject(projectName)
				if err != nil {
					return err
				}
				return visitAll(extractNames(args), func(s string) error {
					logstore, err := project.GetLogStore(s)
					if err != nil {
						return err
					}
					if !o.DryRun {
						if err = logstore.DeleteShipper(s); err != nil {
							return err
						}
					}
					log.GetLogger().Infof("delete logshipper %s success", s)
					return nil
				})
			},
		}
		cmd.AddCommand(deleteCmd)
	}
	cmd.PersistentFlags().StringVarP(&projectName, "project", "p", "", "name of project")
	cmd.MarkPersistentFlagRequired("project")
	return cmd
}
