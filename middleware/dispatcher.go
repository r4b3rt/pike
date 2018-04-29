package custommiddleware

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/mitchellh/go-server-timing"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/vicanso/pike/cache"
	"github.com/vicanso/pike/util"
	"github.com/vicanso/pike/vars"
)

type (
	// DispatcherConfig dipatcher的配置
	DispatcherConfig struct {
		Skipper middleware.Skipper
		// 压缩数据类型
		CompressTypes []string
		// 最小压缩
		CompressMinLength int
		// CompressLevel 数据压缩级别
		CompressLevel int
	}
)

var (
	defaultCompressTypes = []string{
		"text",
		"javascript",
		"json",
	}
)

func save(client *cache.Client, identity []byte, resp *cache.Response, compressible bool) {
	doSave := func() {
		client.SaveResponse(identity, resp)
		client.Cacheable(identity, resp.TTL)
	}
	level := resp.CompressLevel
	compressMinLength := resp.CompressMinLength
	if compressMinLength == 0 {
		compressMinLength = vars.CompressMinLength
	}
	if resp.StatusCode == http.StatusNoContent || !compressible {
		doSave()
		return
	}
	body := resp.Body
	// 如果body为空，但是gzipBody不为空，表示从backend取回来的数据已压缩
	if len(body) == 0 && len(resp.GzipBody) != 0 {
		// 解压gzip数据，用于生成br
		unzipBody, err := util.Gunzip(resp.GzipBody)
		if err != nil {
			doSave()
			return
		}
		body = unzipBody
	}
	bodyLength := len(body)
	// 204没有内容的情况已处理，不应该出现 body为空的现象
	// 如果原始数据还是为空，则直接设置为hit for pass
	if bodyLength == 0 {
		client.HitForPass(identity, vars.HitForPassTTL)
		return
	}
	// 如果数据比最小压缩还小，不需要压缩缓存
	if bodyLength < compressMinLength {
		doSave()
		return

	}
	if len(resp.GzipBody) == 0 {
		gzipBody, _ := util.Gzip(body, level)
		// 如果gzip压缩成功，可以删除原始数据，只保留gzip
		if len(gzipBody) != 0 {
			resp.GzipBody = gzipBody
			resp.Body = nil
		}
	}
	if len(resp.BrBody) == 0 {
		resp.BrBody, _ = util.BrotliEncode(body, level)
	}
	doSave()
	return
}

func shouldCompress(compressTypes []string, contentType string) (compressible bool) {
	for _, v := range compressTypes {
		reg := regexp.MustCompile(v)
		if reg.MatchString(contentType) {
			compressible = true
			return
		}
	}
	return
}

// Dispatcher 对响应数据做缓存，复制等处理
func Dispatcher(config DispatcherConfig, client *cache.Client) echo.MiddlewareFunc {
	compressTypes := config.CompressTypes
	if len(compressTypes) == 0 {
		compressTypes = defaultCompressTypes
	}
	compressMinLength := config.CompressMinLength
	compressLevel := config.CompressLevel
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			status, ok := c.Get(vars.Status).(int)
			if !ok {
				return vars.ErrRequestStatusNotSet
			}
			cr, ok := c.Get(vars.Response).(*cache.Response)
			if !ok {
				return vars.ErrResponseNotSet
			}
			cr.CompressMinLength = compressMinLength
			cr.CompressLevel = compressLevel

			resp := c.Response()
			respHeader := resp.Header()
			reqHeader := c.Request().Header
			timing, _ := c.Get(vars.Timing).(*servertiming.Header)
			var m *servertiming.Metric
			setSeverTiming := func() {
				if m == nil {
					return
				}
				m.Stop()
				for _, m := range timing.Metrics {
					if m.Name == vars.PikeMetric {
						m.Stop()
					}
				}
				respHeader.Add(vars.ServerTiming, timing.String())
			}
			if timing != nil {
				m = timing.NewMetric(vars.DispatchResponseMetric)
				m.WithDesc("dispatch response").Start()
			}
			compressible := shouldCompress(compressTypes, respHeader.Get(echo.HeaderContentType))

			if status == cache.Cacheable {
				// 如果数据是读取缓存，有需要设置Age
				age := uint32(time.Now().Unix()) - cr.CreatedAt
				respHeader.Set(vars.Age, strconv.Itoa(int(age)))
			}

			xStatus := ""
			switch status {
			case cache.Pass:
				xStatus = vars.Pass
			case cache.Fetching:
				xStatus = vars.Fetching
			case cache.HitForPass:
				xStatus = vars.HitForPass
			default:
				xStatus = vars.Cacheable
			}
			respHeader.Set(vars.XStatus, xStatus)
			statusCode := int(cr.StatusCode)

			// pass的都是不可能缓存
			// 可缓存的处理继续后续缓存流程
			if status != cache.Cacheable && status != cache.Pass {
				go func() {
					identity, ok := c.Get(vars.Identity).([]byte)
					if !ok {
						return
					}
					if cr.TTL == 0 {
						client.HitForPass(identity, vars.HitForPassTTL)
					} else {
						save(client, identity, cr, compressible)
					}
				}()
			}

			fresh, _ := c.Get(vars.Fresh).(bool)
			// 304 的处理
			if fresh {
				setSeverTiming()
				resp.WriteHeader(http.StatusNotModified)
				return nil
			}

			acceptEncoding := ""
			// 如果数据不应该被压缩，则直接认为客户端不接受压缩数据
			if compressible {
				acceptEncoding = reqHeader.Get(echo.HeaderAcceptEncoding)
			}
			body, enconding := cr.GetBody(acceptEncoding)
			if enconding != "" {
				respHeader.Set(echo.HeaderContentEncoding, enconding)
			}

			setSeverTiming()
			respHeader.Set(echo.HeaderContentLength, strconv.Itoa(len(body)))
			resp.WriteHeader(statusCode)
			_, err := resp.Write(body)
			return err
		}
	}
}