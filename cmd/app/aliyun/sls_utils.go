package aliyun

import (
	sls "github.com/aliyun/aliyun-log-go-sdk"

	"github.com/fengxsong/toolkit/pkg/log"
)

func createLogstore(client sls.ClientInterface,
	dryRun bool, project string, logstore string, ttl, shardCnt int, autoSplit bool, maxSplitShard int,
	force bool,
	index *sls.Index,
) error {
	exists, err := client.CheckLogstoreExist(project, logstore)
	if err != nil {
		return err
	}
	if !dryRun {
		if !exists {
			log.GetLogger().Debugf("creating logstore %s in project %s", logstore, project)
			if err = client.CreateLogStore(project, logstore, ttl, shardCnt, autoSplit, maxSplitShard); err != nil {
				return err
			}
		} else if force {
			log.GetLogger().Debugf("force updating logstore %s in project %s", logstore, project)
			if err = client.UpdateLogStore(project, logstore, ttl, shardCnt); err != nil {
				return err
			}
		}
		if index != nil {
			log.GetLogger().Debugf("creating index for logstore %s in project %s", logstore, project)
			if err = client.CreateIndex(project, logstore, *index); err != nil {
				if slsErr, ok := err.(*sls.Error); !ok || slsErr.Code != "IndexAlreadyExist" {
					return err
				}
				if force {
					log.GetLogger().Debugf("force updating index for logstore %s in project %s", logstore, project)
					if err = client.UpdateIndex(project, logstore, *index); err != nil {
						return err
					}
				}
			}
		}
	}
	log.GetLogger().Infof("create logstore %s success", logstore)
	return nil
}
