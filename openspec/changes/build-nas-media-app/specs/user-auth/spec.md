# user-auth

用户认证系统。管理员初始化、注册码注册、JWT 登录。所有媒体相关 API 需认证。

## ADDED Requirements

### Requirement: 管理员首次初始化

系统首次启动时 SHALL 检测是否存在管理员账号，若不存在则开放 `/api/auth/setup` 接口，允许创建首个管理员账号。创建成功后该接口 SHALL 永久关闭。

#### Scenario: 首次启动创建管理员

- **WHEN** 系统首次启动且数据库无任何用户
- **THEN** 系统开放 setup 接口，允许以用户名和密码创建管理员账号
- **AND** 创建成功后返回 JWT token，setup 接口不再可用

#### Scenario: 已有管理员时拒绝 setup

- **WHEN** 系统中已存在管理员账号，调用 setup 接口
- **THEN** 系统返回 403 错误

### Requirement: 注册码注册

用户注册 SHALL 必须提供有效注册码。注册码由管理员在配置文件中预置或通过管理接口生成，包含最大使用次数和过期时间。注册成功后注册码使用次数加 1。

#### Scenario: 使用有效注册码注册

- **WHEN** 用户提供有效注册码、用户名和密码调用注册接口
- **AND** 注册码未达到最大使用次数且未过期
- **THEN** 系统创建用户账号，注册码 used 计数 +1，返回 JWT token

#### Scenario: 注册码已用完

- **WHEN** 用户提供的注册码已达到最大使用次数
- **THEN** 系统返回错误提示"注册码已用完，请联系管理员"

#### Scenario: 注册码已过期

- **WHEN** 用户提供的注册码已超过过期时间
- **THEN** 系统返回错误提示"注册码已过期"

#### Scenario: 无效注册码

- **WHEN** 用户提供的注册码不在配置列表中
- **THEN** 系统返回错误提示"无效的注册码"

#### Scenario: 用户名重复

- **WHEN** 注册码有效但用户名已被占用
- **THEN** 系统返回错误提示"用户名已存在"

### Requirement: JWT 登录

用户 SHALL 通过用户名和密码登录获取 JWT token，token 有效期 7 天。所有 `/api/stream/*` 和受保护的 API 路由需 Bearer token 验证。

#### Scenario: 正确凭据登录

- **WHEN** 用户提供正确的用户名和密码
- **THEN** 系统返回 JWT token，有效期为 7 天

#### Scenario: 错误凭据拒绝

- **WHEN** 用户提供错误的密码
- **THEN** 系统返回 401 "用户名或密码错误"

### Requirement: 认证中间件保护媒体路由

所有媒体流、媒体管理、搜索、播放列表等 API SHALL 通过 JWT 认证中间件保护。未携带有效 token 的请求 SHALL 返回 401。

#### Scenario: 未认证访问受保护 API

- **WHEN** 请求未携带 Authorization header 访问 `/api/media`
- **THEN** 系统返回 401

#### Scenario: 过期 token 访问

- **WHEN** 请求携带已过期的 JWT token
- **THEN** 系统返回 401 "token 已过期"

#### Scenario: 有效 token 访问

- **WHEN** 请求携带有效的 JWT token 访问受保护 API
- **THEN** 系统正常处理请求并返回数据

### Requirement: TV 设备码登录

系统 SHALL 支持 TV 端通过设备码（Device Code）方式登录，避免遥控器输入密码。TV 端请求设备码后，用户在已登录的手机或 PC 端输入该码完成授权。

#### Scenario: TV 端请求设备码

- **WHEN** TV 端调用 `/api/auth/device-code`
- **THEN** 系统生成一个 4 位字母数字设备码，有效期 10 分钟

#### Scenario: 用户在其他设备输入设备码授权

- **WHEN** 已登录用户在手机/PC 端输入有效的设备码
- **THEN** 系统将设备码与当前用户关联，TV 端轮询获得 JWT token

#### Scenario: 设备码过期

- **WHEN** 设备码超过 10 分钟未被授权
- **THEN** 设备码失效，TV 端需重新请求
