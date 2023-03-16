package download

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"github.com/qiniu/qshell/v2/iqshell/common/alert"
	"github.com/qiniu/qshell/v2/iqshell/common/config"
	"github.com/qiniu/qshell/v2/iqshell/common/data"
	"github.com/qiniu/qshell/v2/iqshell/common/flow"
	"github.com/qiniu/qshell/v2/iqshell/common/host"
	"github.com/qiniu/qshell/v2/iqshell/common/utils"
	"github.com/qiniu/qshell/v2/iqshell/common/workspace"
	"github.com/qiniu/qshell/v2/iqshell/storage/bucket"
	"net/url"
	"strings"
	"time"
)

type UrlApiInfo struct {
	BucketDomain string
	Key          string
	UseHttps     bool
}

// PublicUrl 返回公有空间的下载链接，不可以用于私有空间的下载
func PublicUrl(info UrlApiInfo) (fileUrl string) {
	domain := utils.RemoveUrlScheme(info.BucketDomain)
	if info.UseHttps {
		fileUrl = fmt.Sprintf("https://%s/%s", domain, url.PathEscape(info.Key))
	} else {
		fileUrl = fmt.Sprintf("http://%s/%s", domain, url.PathEscape(info.Key))
	}
	return
}

// PublicUrlToPrivateApiInfo 私有下载链接
type PublicUrlToPrivateApiInfo struct {
	PublicUrl string
	Deadline  int64
}

type PublicUrlToPrivateApiResult struct {
	Url string
}

var _ flow.Result = (*PublicUrlToPrivateApiResult)(nil)

func (p *PublicUrlToPrivateApiResult) IsValid() bool {
	return len(p.Url) > 0
}

// PublicUrlToPrivate 公转私
func PublicUrlToPrivate(info PublicUrlToPrivateApiInfo) (result *PublicUrlToPrivateApiResult, err *data.CodeError) {
	if len(info.PublicUrl) == 0 {
		return nil, alert.CannotEmptyError("url", "")
	}

	if info.Deadline < 1 {
		return nil, data.NewEmptyError().AppendDesc("deadline is invalid")
	}

	m, err := bucket.GetBucketManager()
	if err != nil {
		return nil, err
	}

	srcUri, pErr := url.Parse(info.PublicUrl)
	if pErr != nil {
		err = data.ConvertError(pErr)
		return
	}

	h := hmac.New(sha1.New, m.Mac.SecretKey)

	urlToSign := srcUri.String()
	if strings.Contains(info.PublicUrl, "?") {
		urlToSign = fmt.Sprintf("%s&e=%d", urlToSign, info.Deadline)
	} else {
		urlToSign = fmt.Sprintf("%s?e=%d", urlToSign, info.Deadline)
	}
	h.Write([]byte(urlToSign))

	sign := base64.URLEncoding.EncodeToString(h.Sum(nil))
	token := m.Mac.AccessKey + ":" + sign
	return &PublicUrlToPrivateApiResult{
		Url: fmt.Sprintf("%s&token=%s", urlToSign, token),
	}, nil
}

// PrivateUrl 返回私有空间的下载链接， 也可以用于公有空间的下载
func PrivateUrl(info UrlApiInfo) (fileUrl string) {
	publicUrl := PublicUrl(UrlApiInfo(info))
	deadline := time.Now().Add(time.Hour * 24 * 30).Unix()
	result, _ := PublicUrlToPrivate(PublicUrlToPrivateApiInfo{
		PublicUrl: publicUrl,
		Deadline:  deadline,
	})
	if result != nil {
		fileUrl = result.Url
	}
	return
}

// 下载 Url
func createDownloadUrl(info *DownloadActionInfo, useHttps bool) (string, *data.CodeError) {
	h, hErr := info.HostProvider.Provide()
	if hErr != nil {
		return "", hErr.HeaderInsertDesc("[provide host]")
	}
	return createDownloadUrlWithHost(h, info, useHttps)
}

func createDownloadUrlWithHost(h *host.Host, info *DownloadActionInfo, useHttps bool) (string, *data.CodeError) {
	urlString := ""
	server := h.GetServer()

	// 构造下载 url
	if info.UseGetFileApi {
		mac, err := workspace.GetMac()
		if err != nil {
			return "", data.NewEmptyError().AppendDescF("download get mac error:%v", mac)
		}
		urlString = utils.Endpoint(useHttps, server)
		urlString = strings.Join([]string{urlString, "getfile", mac.AccessKey, info.Bucket, url.PathEscape(info.Key)}, "/")
	} else {
		isSrcDomain := isIoSrcHost(server)
		// 源站域名需要签名
		if info.IsPublic && !isSrcDomain {
			urlString = PublicUrl(UrlApiInfo{
				BucketDomain: server,
				Key:          info.Key,
				UseHttps:     useHttps,
			})
		} else {
			urlString = PrivateUrl(UrlApiInfo{
				BucketDomain: server,
				Key:          info.Key,
				UseHttps:     useHttps,
			})
		}
	}
	return urlString, nil
}

// CreateSrcDownloadDomainWithBucket 公有云 bucket 源站下载域名
func CreateSrcDownloadDomainWithBucket(cfg *config.Config, bucketName string, regionId string) (string, *data.CodeError) {

	ioSrcHost := ""
	if cfg != nil {
		ioSrcHost = cfg.GetIoSrcHost()
	}
	if len(ioSrcHost) == 0 {
		ioSrcHost = createPublicCloudSrcDownloadEndPoint(regionId)
	}
	return bucketName + "." + ioSrcHost, nil
}

func isIoSrcHost(host string) bool {
	customEndpoint := ""
	if workspace.GetConfig() != nil {
		customEndpoint = workspace.GetConfig().GetIoSrcHost()
	}
	if len(customEndpoint) > 0 {
		return strings.Contains(host, customEndpoint)
	} else {
		return isPublicCloudSrcDownloadDomain(host)
	}
}

func isPublicCloudSrcDownloadDomain(domain string) bool {
	return strings.Contains(domain, "kodo-") && strings.HasSuffix(domain, ".qiniucs.com")
}

func createPublicCloudSrcDownloadEndPoint(regionId string) string {
	return "kodo-" + regionId + ".qiniucs.com"
}
