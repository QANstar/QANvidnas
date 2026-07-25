# file-upload

文件上传。PC Web 端上传音视频文件到 NAS 指定目录。

## ADDED Requirements

### Requirement: Web 端文件上传

PC 端用户 SHALL 能通过 Web 界面上传音视频文件到管理员配置的扫描文件夹中。支持大文件上传（最大 10GB）。

#### Scenario: 上传视频文件

- **WHEN** 用户在 PC 端选择视频文件（如 MP4）上传
- **AND** 文件大小在限制内
- **THEN** 系统接收文件并保存到目标目录
- **AND** 展示上传进度

#### Scenario: 上传大文件显示进度

- **WHEN** 用户上传超过 100MB 的文件
- **THEN** 前端显示进度条（百分比和速度）

#### Scenario: 文件大小超限

- **WHEN** 用户选择超过 10GB 的文件
- **THEN** 前端在上传前提示"文件过大，最大支持 10GB"

#### Scenario: 上传后自动扫描

- **WHEN** 文件上传完成
- **THEN** 文件进入扫描队列，自动被索引和生成缩略图

### Requirement: 目标目录选择

用户上传时 SHALL 能从管理员配置的扫描文件夹中选择目标目录。默认选择最近使用的目录。

#### Scenario: 选择目标目录

- **WHEN** 用户进入上传页面
- **THEN** 显示可选的目标目录列表（来自管理员配置的扫描文件夹及其子目录）

### Requirement: 格式限制

上传接口 SHALL 仅接受支持的音视频格式，文件扩展名校验。

#### Scenario: 上传支持格式

- **WHEN** 用户上传 MP4/MKV/AVI/MP3 等支持格式的文件
- **THEN** 系统接受上传

#### Scenario: 上传不支持格式

- **WHEN** 用户选择非音视频格式文件（如 .exe, .zip）
- **THEN** 前端在上传前提示"不支持的格式"，阻止上传
