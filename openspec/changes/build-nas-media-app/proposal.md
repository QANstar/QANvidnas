## Why

NAS 上存了大量音视频文件，但没有一个方便的多端管理播放工具。现有的开源方案（Jellyfin/Emby）太重、配置复杂，且界面不够现代活泼。QANvidnas 要做的是一个轻量、美观、开箱即用的 NAS 媒体中心——部署简单（Docker 双容器），支持手机/电视/PC 三端，用注册码控制访问。

## What Changes

- 新增 ASP.NET Core 9.0 后端服务，提供 REST API、媒体流、文件监听、缩略图生成等全部服务端能力
- 新增 React 前端（Web），独立部署在 Nginx 容器，支持 PC 浏览器和手机浏览器访问
- 新增 Capacitor 包装的 Android APK（手机端），支持原生 ExoPlayer 播放和后台音频
- 新增 Capacitor 包装的 Android TV APK（电视端），支持原生 ExoPlayer 硬解和遥控器交互
- 新增用户系统：管理员初始化 → 注册码注册 → JWT 认证，所有媒体文件需登录后才能访问
- 新增 NAS 文件夹扫描：管理员指定路径 → 自动/手动扫描 → 提取元数据
- 新增媒体元数据管理：名称、描述、标签的自定义编辑
- 新增标签筛选与全文搜索（LIKE 查询，文件名、标签、描述）
- 新增文件夹级播放列表：一键播放文件夹及其子文件夹内所有媒体
- 新增播放模式：顺序、循环、随机(Fisher-Yates洗牌)、单曲循环
- 新增封面系统：自动取视频 10% 位置帧 + 黑场检测回退 + 自定义上传封面
- 新增视频播放进度条缩略图预览（FFmpeg 雪碧图，每 10s 一帧）
- 新增 TV 端长按快进渐进加速曲线（解决长视频遥控器快进困难）
- 新增 PC 端文件上传到 NAS
- 新增粉蓝活泼配色的响应式 UI，适配手机/PC/TV 三种屏幕

## Capabilities

### New Capabilities

- `user-auth`: 用户认证系统。管理员初始化设置、注册码注册、JWT 登录、所有媒体 API 需认证访问。
- `media-scan`: 媒体扫描与索引。管理员指定 NAS 文件夹路径，手动扫描，提取视频/音频文件元数据（v1 使用手动触发 + 定时轮询）。
- `media-metadata`: 媒体元数据管理。编辑名称、描述、标签，自定义封面上传，自动封面生成(FFmpeg 取帧+黑场检测)。
- `media-search`: 媒体搜索与筛选。按标签筛选，全文搜索(文件名/标签/描述)，EF Core LINQ Contains 查询。
- `media-stream`: 媒体流服务。HTTP Range 请求支持视频/音频流，所有文件通过 .NET 后端代理(需认证)。
- `playlist`: 播放列表管理。文件夹即播放列表，一键递归播放，顺序/循环/随机/单曲循环四种模式，Fisher-Yates 洗牌算法。
- `tv-remote-control`: 电视端遥控器交互。渐进加速长按快进(5级跳度曲线)、D-pad 焦点导航、遥控器媒体键处理。
- `thumbnail-generation`: 缩略图生成管线。FFmpeg 封面取帧(10%位置+黑场回退)、进度条雪碧图(每10s一帧)。
- `file-upload`: 文件上传。PC Web 端上传音视频到 NAS 指定目录。
- `config-management`: 配置管理。appsettings.json + config.yaml 配置文件，样例文件提交 git，真实配置 gitignore。

### Modified Capabilities

<!-- No existing capabilities to modify - greenfield project -->

## Impact

- **新项目**：全部代码从零构建，无现有代码影响
- **依赖**：.NET 9.0, React 19+, SQLite (EF Core), FFmpeg, Capacitor 6, ExoPlayer, Nginx
- **部署**：Docker 双容器部署在飞牛 NAS (x86)，HTTP 协议局域网访问，NAS 媒体文件夹通过 Docker volume 挂载
- **构建产物**：.NET 服务器 Docker 镜像 + Nginx 前端 Docker 镜像 + Android 手机 APK + Android TV APK
