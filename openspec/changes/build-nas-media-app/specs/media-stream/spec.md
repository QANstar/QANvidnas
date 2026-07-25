# media-stream

媒体流服务。通过 Go 后端代理所有音视频文件访问，支持 HTTP Range 请求，所有请求需认证。

## ADDED Requirements

### Requirement: HTTP Range 流传输

音视频文件 SHALL 通过 Go HTTP Handler 代理传输，完整支持 HTTP Range 请求（RFC 7233），实现客户端 seek 操作。

#### Scenario: 请求视频流（带 Range 头）

- **WHEN** 客户端发送带有 `Range: bytes=0-1048575` 头的请求
- **THEN** 系统返回 206 Partial Content，Content-Range 响应头正确指示返回的字节范围

#### Scenario: 请求完整文件（无 Range 头）

- **WHEN** 客户端发送不带 Range 头的请求
- **THEN** 系统返回 200 OK，传输完整文件

#### Scenario: Range 超出文件范围

- **WHEN** 客户端请求的 Range 起始位置超出文件大小
- **THEN** 系统返回 416 Range Not Satisfiable

### Requirement: 认证保护

所有 `/api/stream/*` 路由 SHALL 要求有效的 JWT token，未认证请求返回 401。

#### Scenario: 未认证访问流

- **WHEN** 请求未携带有效 JWT token 访问流接口
- **THEN** 系统返回 401 而非重定向（API 接口不返回 HTML）

### Requirement: 文件路径安全

流服务 SHALL 验证请求的媒体 ID 对应的文件路径在配置的允许范围内，防止路径遍历攻击。

#### Scenario: 正常路径访问

- **WHEN** 请求有效的媒体 ID，其文件路径在扫描文件夹范围内
- **THEN** 系统正常读取并传输文件

#### Scenario: 路径遍历防护

- **WHEN** 媒体记录的路径包含 `..` 或符号链接指向扫描目录外（极端情况）
- **THEN** 系统拒绝访问，返回 403

### Requirement: MIME 类型识别

系统 SHALL 根据文件扩展名设置正确的 Content-Type 响应头，确保客户端正确识别音频或视频流。

#### Scenario: 视频文件 MIME

- **WHEN** 请求 MP4 视频流
- **THEN** 系统返回 `Content-Type: video/mp4`

#### Scenario: 音频文件 MIME

- **WHEN** 请求 MP3 音频流
- **THEN** 系统返回 `Content-Type: audio/mpeg`
