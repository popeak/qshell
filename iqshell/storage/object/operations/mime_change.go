package operations

import (
	"github.com/qiniu/qshell/v2/iqshell"
	"github.com/qiniu/qshell/v2/iqshell/common/alert"
	"github.com/qiniu/qshell/v2/iqshell/common/group"
	"github.com/qiniu/qshell/v2/iqshell/common/log"
	"github.com/qiniu/qshell/v2/iqshell/common/utils"
	"github.com/qiniu/qshell/v2/iqshell/storage/object"
	"github.com/qiniu/qshell/v2/iqshell/storage/object/batch"
)

type ChangeMimeInfo object.ChangeMimeApiInfo

func (info *ChangeMimeInfo) Check() error {
	if len(info.Bucket) == 0 {
		return alert.CannotEmptyError("Bucket", "")
	}
	if len(info.Key) == 0 {
		return alert.CannotEmptyError("Key", "")
	}
	if len(info.Mime) == 0 {
		return alert.CannotEmptyError("MimeType", "")
	}
	return nil
}

func ChangeMime(cfg *iqshell.Config, info ChangeMimeInfo) {
	if shouldContinue := iqshell.CheckAndLoad(cfg, iqshell.CheckAndLoadInfo{
		Checker: &info,
	}); !shouldContinue {
		return
	}

	result, err := object.ChangeMimeType(object.ChangeMimeApiInfo(info))
	if err != nil {
		log.ErrorF("Change Mime error:%v", err)
		return
	}

	if len(result.Error) != 0 {
		log.ErrorF("Change Mime result error:%v", result.Error)
		return
	}
}

type BatchChangeMimeInfo struct {
	BatchInfo batch.Info
	Bucket    string
}

func (info *BatchChangeMimeInfo) Check() error {
	if err := info.BatchInfo.Check(); err != nil {
		return err
	}

	if len(info.Bucket) == 0 {
		return alert.CannotEmptyError("Bucket", "")
	}
	return nil
}

func BatchChangeMime(cfg *iqshell.Config, info BatchChangeMimeInfo) {
	if shouldContinue := iqshell.CheckAndLoad(cfg, iqshell.CheckAndLoadInfo{
		Checker: &info,
	}); !shouldContinue {
		return
	}

	handler, err := group.NewHandler(info.BatchInfo.Info)
	if err != nil {
		log.Error(err)
		return
	}
	batch.NewFlow(info.BatchInfo).ReadOperation(func() (operation batch.Operation, hasMore bool) {
		line, success := handler.Scanner().ScanLine()
		if !success {
			return nil, false
		}

		items := utils.SplitString(line, info.BatchInfo.ItemSeparate)
		if len(items) > 1 {
			key, mime := items[0], items[1]
			if key != "" && mime != "" {
				return object.ChangeMimeApiInfo{
					Bucket: info.Bucket,
					Key:    key,
					Mime:   mime,
				}, true
			}
		}
		return nil, true
	}).OnResult(func(operation batch.Operation, result batch.OperationResult) {
		apiInfo, ok := (operation).(object.ChangeMimeApiInfo)
		if !ok {
			return
		}
		in := ChangeMimeInfo(apiInfo)
		if result.Code != 200 || result.Error != "" {
			handler.Export().Fail().ExportF("%s\t%s\t%d\t%s", in.Key, in.Mime, result.Code, result.Error)
			log.ErrorF("Chgm '%s' => '%s' Failed, Code: %d, Error: %s",
				in.Key, in.Mime, result.Code, result.Error)
		} else {
			handler.Export().Success().ExportF("%s\t%s", in.Key, in.Mime)
			log.InfoF("Chgm '%s' => '%s' success", in.Key, in.Mime)
		}
	}).OnError(func(err error) {
		log.ErrorF("batch chgm error:%v:", err)
	}).Start()
}
