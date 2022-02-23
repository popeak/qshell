package operations

import (
	"bufio"
	"github.com/qiniu/qshell/v2/iqshell"
	"github.com/qiniu/qshell/v2/iqshell/common/alert"
	"os"

	"github.com/qiniu/qshell/v2/iqshell/common/log"
	"github.com/qiniu/qshell/v2/iqshell/common/utils"
)

type RpcInfo struct {
	Params []string
}

func (info *RpcInfo) Check() error {
	return nil
}

func RpcDecode(cfg *iqshell.Config, info RpcInfo) {
	if shouldContinue := iqshell.CheckAndLoad(cfg, iqshell.CheckAndLoadInfo{
		Checker: &info,
	}); !shouldContinue {
		return
	}

	if len(info.Params) > 0 {
		for _, param := range info.Params {
			decodedStr, _ := utils.Decode(param)
			log.Alert(decodedStr)
		}
	} else {
		bScanner := bufio.NewScanner(os.Stdin)
		for bScanner.Scan() {
			toDecode := bScanner.Text()
			decodedStr, _ := utils.Decode(string(toDecode))
			log.Alert(decodedStr)
		}
	}
}

func RpcEncode(cfg *iqshell.Config, info RpcInfo) {
	if shouldContinue := iqshell.CheckAndLoad(cfg, iqshell.CheckAndLoadInfo{
		Checker: &info,
	}); !shouldContinue {
		return
	}

	if len(info.Params) == 0 {
		log.Error(alert.CannotEmpty("rpc encode Value", ""))
		return
	}
	for _, param := range info.Params {
		encodedStr := utils.Encode(param)
		log.Alert(encodedStr)
	}
}
