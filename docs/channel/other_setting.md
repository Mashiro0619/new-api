# 渠道额外设置说明

该配置用于设置额外的渠道参数，可以通过 JSON 对象进行配置。

1. force_format
    - 旧版 OpenAI 响应修整开关，只处理响应格式，不改变请求的出站协议
    - 类型为布尔值，设置为 true 时启用强制格式化

2. forced_outbound_format
    - 强制把支持的文本生成请求转换成指定协议后再发送给上游，响应仍会转换回客户端的入站协议
    - 仅支持 OpenAI（类型 1）和 New API（类型 60）渠道
    - 可选值为 `openai`、`openai_responses`、`claude`、`gemini`；不设置或留空表示继续跟随渠道类型
    - 只作用于 Chat Completions、Responses、Anthropic Messages 和 Gemini GenerateContent；图片、音频、Embedding、Rerank、Realtime、任务等接口保持原逻辑
    - 启用后优先于全局或渠道级请求体透传，也优先于 Chat Completions 自动转 Responses
    - 上游 Base URL 必须支持所选协议的标准路径；Claude、Gemini 等协议互转可能丢弃无法映射的协议专属字段

3. proxy
    - 用于配置网络代理
    - 类型为字符串，支持 `http`、`https`、`socks5` 和 `socks5h` 协议
    - 保存时必须包含协议和主机；仅允许空路径或根路径 `/`，不允许 query 或 fragment
    - SOCKS 代理未填写端口时，运行时使用默认端口 `1080`

4. thinking_to_content
   - 用于标识是否将思考内容`reasoning_content`转换为`<think>`标签拼接到内容中返回
   - 类型为布尔值，设置为 true 时启用思考内容转换

--------------------------------------------------------------

## JSON 格式示例

以下示例把该渠道的文本请求统一转换成 OpenAI Responses 协议，并设置代理地址：

```json
{
    "forced_outbound_format": "openai_responses",
    "force_format": false,
    "thinking_to_content": true,
    "pass_through_body_enabled": false,
    "proxy": "socks5://proxy.example:1080"
}
```

--------------------------------------------------------------

渠道编辑页选择“跟随渠道类型”时不会写入 `forced_outbound_format`。启用强制出站协议后，前端会主动关闭渠道级请求体透传；若通过 API 保存了两项同时启用的配置，运行时仍以强制出站协议为准。

## 升级兼容性

旧版本会忽略代理地址中的 path、query 和 fragment。为避免升级后中断已有渠道流量，运行时会继续剥离这些遗留后缀，并对同一代理地址每个进程记录一次不含凭证和后缀的警告。该兼容逻辑不会改写数据库；再次保存渠道时必须按上述严格规则修正代理地址。

代理连接使用 30 秒 TCP 拨号超时和 30 秒 KeepAlive；TLS 握手超时为 10 秒。这些超时同样适用于未配置渠道代理的中转请求。
