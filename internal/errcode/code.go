package errcode

// 全局返回码
// 0   = 成功
// 负数 = 系统错误（go-zero 默认）
// 正数 = 业务错误码

const (
	// 通用 1000-1999
	BadRequest   = 1000
	InvalidParam = 1001

	// 文档模块 2000-2999
	DocNotFound         = 2001
	DocTypeNotSupported = 2002
	DocUploadFailed     = 2003
)

var codeMsg = map[int]string{
	Success:             "success",
	BadRequest:          "请求参数错误",
	InvalidParam:        "参数校验失败",
	DocNotFound:         "文档不存在",
	DocTypeNotSupported: "不支持的文档类型",
	DocUploadFailed:     "文档上传失败",
}

const Success = 0

func Msg(code int) string {
	if msg, ok := codeMsg[code]; ok {
		return msg
	}
	return "未知错误"
}
