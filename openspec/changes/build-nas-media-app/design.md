## Context

QANvidnas 是一个自托管的 NAS 媒体管理与播放应用。部署在飞牛 NAS (x86) 的 Docker 环境，为家庭用户提供手机/电视/PC 三端统一的音视频浏览、搜索和播放体验。项目从零构建，无历史代码负担。

**约束条件：**
- Docker 部署，优先考虑部署简易性
- 服务端 .NET 9.0，前端 React + Vite（团队技术偏好）
- Android TV 体验要求原生级别（硬解、遥控器交互）
- 手机端需支持后台音频播放
- HTTP 局域网通信（v1 不要求 HTTPS）
- 所有媒体访问需要登录认证

## Goals / Non-Goals

**Goals:**
- .NET 9.0 后端 + Nginx 前端双容器部署，docker-compose 一键编排
- Capacitor 包装 Android APK（手机+TV），WebView 加载远程 UI，原生 ExoPlayer 处理播放
- 注册码用户体系，管理员控制谁可以访问
- 文件夹级媒体库管理，标签+全文搜索
- 电视端渐进加速快进，解决长视频遥控器操作痛点
- 自动封面生成（FFmpeg）+ 进度条雪碧图预览
- 前端 API 地址通过 .env 配置文件管理，重新构建即可切换后端地址

**Non-Goals:**
- 不做转码/实时转码（直接流传输原文件，客户端硬解）
- 不做字幕支持
- v1 不做 HTTPS（后续用 Caddy sidecar 或用户自配反向代理）
- 不做 DLNA/UPnP 协议支持
- 不做多语言国际化（仅中文）
- 不做 iOS 客户端（v1 仅 Android）
- 不做 ISO 原盘/蓝光结构解析

## Decisions

### D1: 前后端分离部署 vs Go 单体内嵌

**选择：前后端分离，.NET 后端 + Nginx 前端双容器部署。**

理由：v1 初期采用 Go 单体内嵌方案，但在飞牛 NAS 上遇到 CGO + musl libc 版本不匹配导致网络层面异常，排查困难。换成 .NET 9.0 后与另一已验证项目 QANassistant_Server 采用相同技术栈，部署经验可复用。前后端分离虽然增加一个容器，但各自职责清晰、独立扩展、故障隔离。Nginx 反向代理 /api 到后端，同时提供 SPA 静态文件服务。

替代方案：Go 单体内嵌——在飞牛 NAS Docker 环境下 CGO 编译的静态二进制存在 musl 兼容性风险，已被否决。

### D2: SQLite + EF Core vs PostgreSQL

**选择：SQLite + Entity Framework Core，LIKE 查询替代 FTS5。**

理由：家庭场景并发 < 10，SQLite 完全胜任。.NET 的 EF Core SQLite 驱动原生且不需要 CGO，无跨平台兼容问题。放弃 FTS5 全文搜索引擎（需要 C 扩展），改用 EF Core LINQ Contains 做模糊搜索，对 NAS 家庭场景的媒体量级（通常 < 10 万文件）性能足够（< 200ms）。

替代方案：PostgreSQL——需要独立容器，对家庭场景过度设计。SQLite FTS5——需要 CGO session 扩展，增加跨平台部署风险。

### D3: Capacitor vs React Native vs 纯 WebView

**选择：Capacitor + 自定义 ExoPlayer 插件。**

理由：
- Capacitor 是标准化的 WebView 壳方案，不改变 React 代码结构
- 自定义 Plugin 机制让原生 ExoPlayer 与 JS 层通过类型化接口通信
- 手机和 TV 共享同一个 Capacitor 工程，仅平台配置不同
- React 代码 95%+ 复用，平台差异仅限播放器桥接层

替代方案：React Native——需重写全部 UI 层，TV 支持不成熟。纯 WebView——JSBridge 需手写协议，不如 Capacitor Plugin API 规范。

### D4: TV 端 ExoPlayer 原生播放 vs HLS/DASH 自适应流

**选择：ExoPlayer 原生播放，.NET 后端直接 serve 原始文件 + HTTP Range。**

理由：NAS 局域网带宽充足（通常 1Gbps），无需自适应码率。ExoPlayer 直接解码 MKV/MP4/AVI 等原始格式，零转码开销。.NET Stream Handler 仅处理 Range 请求实现 seek。

替代方案：实时转码 HLS——CPU 消耗巨大，NAS 性能不足以实时转码高码率视频，且增加延迟。

### D5: 电视快进：渐进加速曲线

**选择：5 级跳度，时间阈值触发加速。**

```
级数 | 按住时长 | 跳度 | 频率   | 实际速度
L1  | 0-2s    | 5s   | 3次/s | 15s/按住秒
L2  | 2-5s    | 10s  | 3次/s | 30s/按住秒
L3  | 5-8s    | 30s  | 4次/s | 120s/按住秒
L4  | 8-12s   | 60s  | 4次/s | 240s/按住秒
L5  | 12s+    | 300s | 5次/s | 1500s/按住秒
```

2 小时电影从头到尾约需 12-14 秒，开头精度高（5s 跳度），末尾速度快（5min 跳度）。

替代方案：固定速度快进——长视频体验极差。YouTube 式双击跳 10s——电视遥控器无触屏。

### D6: 封面生成：10% 位置取帧 + 黑场检测

**选择：`ffmpeg -ss <10%时长> -vframes 1`，计算平均亮度，< 30/255 判定为黑场，回退到 30% 位置。**

理由：10% 位置跳过片头但不过深，业界通用做法（Jellyfin/Emby 类似）。黑场检测避免纯黑封面。

替代方案：固定第 N 秒——不同视频片头长短差异大，不可靠。取多帧让用户选——增加 FFmpeg 调用次数和用户操作负担。

### D7: 文件监听：手动扫描 + 定时轮询

**选择：v1 使用手动触发全量扫描 + 5 分钟轮询增量检查。后续版本可增加 FileSystemWatcher 实时监听。**

理由：.NET 的 FileSystemWatcher 在 Docker 挂载卷场景下同样存在可靠性问题（与 fsnotify 类似）。先提供手动扫描和定时轮询作为可靠基线，后续根据用户反馈决定是否增加实时监听。

### D8: 随机播放：Fisher-Yates 洗牌

**选择：播放列表加载时执行 Fisher-Yates 洗牌，生成随机排列后顺序播放。切换到"下一个"沿洗牌后列表前进，保证不重复。**

理由：避免"每次随机取一首"可能导致的短期重复问题。用户切完所有歌之前每首只播一次。

### D9: Monorepo 单仓库

**选择：`server/` (.NET) + `client/` (React+Capacitor) 同仓库，docker-compose 统一编排。**

理由：React 源码是三端的唯一 UI 来源；Capacitor APK 加载同一份 React 源码构建的远程 UI。拆分仓库会增加构建产物传递的复杂度（需 CI artifact pipeline 或 npm 私有包），对当前项目规模是过度工程化。

### D10: .env 构建时前端配置

**选择：Vite 标准 .env 文件管理前端 API 地址，构建时注入 `import.meta.env.VITE_API_BASE_URL`。**

理由：.env 是 Vite 原生支持的配置方式，TypeScript 类型安全，构建时注入零运行时开销。部署后如需切换后端地址，修改 .env 后重新 `docker compose build frontend` 即可。`.env.example` 提交 git 作为模板，`.env` 加入 .gitignore。

替代方案：运行时 config.js——需额外 script 标签和 window 全局变量，不如 .env 标准化。

## Risks / Trade-offs

- **[R1] Docker 挂载卷的文件监听可能不可靠** → v1 使用手动扫描 + 5 分钟轮询。管理面板显示"上次扫描时间"让用户感知状态。
- **[R2] Android TV WebView 性能差异大** → 播放器走原生 ExoPlayer，WebView 仅负责 UI。UI 端避免复杂动画，使用 CSS transform 而非 JS 动画。低端设备降低雪碧图帧数。
- **[R3] FFmpeg 依赖** → .NET 后端 Docker 镜像基于 Debian（`mcr.microsoft.com/dotnet/aspnet:9.0`），通过 apt-get 安装 ffmpeg。封面/缩略图生成异步执行，不阻塞扫描主流程。
- **[R4] .NET Docker 镜像体积** → aspnet:9.0 + ffmpeg 约 300MB，比 Alpine Go 二进制大，但消除了 CGO/libc 兼容性风险。可接受的家庭 NAS 场景。
- **[R5] Capacitor Plugin 原生 ExoPlayer 与 WebView 的 SurfaceView 层级** → SurfaceView 默认在最上层，可能遮挡 WebView 弹窗。播放控制 UI 由 React 渲染在 WebView 中，需调整布局避免被遮挡区域。

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    飞牛 NAS (Docker)                              │
│                                                                  │
│  ┌──────────────────────┐    ┌──────────────────────────────┐   │
│  │  frontend (Nginx)    │    │  backend (.NET 9.0)           │   │
│  │  port: 8080 → 80     │    │  port: 6666                   │   │
│  │                      │    │                               │   │
│  │  React SPA (dist/)   │    │  ┌─────────────────────────────┐ │
│  │  SPA fallback        │    │  │ Auth /api/auth/*           │ │
│  │  /api → proxy_pass   │────│→ │ Media /api/media/*         │ │
│  │      to backend:6666 │    │  │ Stream /api/stream/*       │ │
│  │                      │    │  │ Search /api/search + /tags │ │
│  │  gzip                │    │  │ Playlist /api/playlists/*  │ │
│  │                      │    │  │ Upload /api/upload         │ │
│  └──────────────────────┘    │  │                             │ │
│                              │  │ ┌─────────────────────────┐ │ │
│                              │  │ │ Scanner (ffprobe)       │ │ │
│                              │  │ │ Thumbnail (ffmpeg)      │ │ │
│                              │  │ └─────────────────────────┘ │ │
│                              │  │                             │ │
│                              │  │ ┌─────────────────────────┐ │ │
│                              │  │ │ SQLite (EF Core)        │ │ │
│                              │  │ │ /app/data/qanvidnas.db  │ │ │
│                              │  │ └─────────────────────────┘ │ │
│                              │  └─────────────────────────────┘ │
│                              └──────────────────────────────────┘
│        │                           │
│   ┌────┴────┐                 ┌────┴────┐
│   │ /media  │                 │ /config │  ← Docker volume mounts
│   │ (NAS)   │                 │ .yaml   │
│   └─────────┘                 │ /storage│
│                               └─────────┘
└─────────────────────────────────────────────────────────────────┘
```

## Data Model

```
User: id (long), username, password_hash, is_admin, created_at
InviteCode: code, description, max_uses, used, expires_at
Media: id (long), title, description, type(video/audio), path, duration,
       resolution, cover_path, sprite_path, sprite_meta, file_size, codec,
       bitrate, deleted, created_at, updated_at
Tag: id (long), name, color
MediaTag: media_id, tag_id (M:N)
Playlist: id (long), name, folder_path, user_id, play_mode, created_at
PlaylistItem: playlist_id, media_id, position
ScanFolder: id (long), path, last_scan_at, status
DeviceCode: code, user_id, expires_at, used
```

## API Route Design

```
# Auth
GET    /api/auth/check-setup         # 检查是否需要初始化
POST   /api/auth/setup               # 管理员初始化（仅首次）
POST   /api/auth/login               # 登录 → JWT
POST   /api/auth/register            # 注册（需注册码）
GET    /api/auth/me                  # 当前用户信息
GET    /api/auth/device-code         # 生成 TV 设备码
GET    /api/auth/device-code/poll    # 轮询设备码授权状态
POST   /api/auth/device-code/authorize  # 授权设备码
POST   /api/auth/change-password     # 修改密码

# Media
GET    /api/media                    # 媒体列表（分页、筛选、排序）
GET    /api/media/:id                # 媒体详情
PUT    /api/media/:id                # 编辑元数据（名称、描述、标签）
POST   /api/media/:id/cover          # 上传自定义封面
DELETE /api/media/:id/cover          # 删除封面恢复自动
GET    /api/media/:id/sprite         # 获取雪碧图元数据
POST   /api/media/scan               # 手动触发扫描
GET    /api/media/scan/progress      # 获取扫描进度

# Stream
GET    /api/stream/video/:id         # 视频流（Range 支持）
GET    /api/stream/audio/:id         # 音频流（Range 支持）
GET    /api/stream/cover/:id         # 封面图 / 雪碧图

# Search
GET    /api/search?q=&tags=&type=    # 搜索 + 标签筛选

# Tags
GET    /api/tags                     # 所有标签列表
POST   /api/tags                     # 创建标签
DELETE /api/tags/:id                 # 删除标签

# Playlist
GET    /api/playlists                # 用户播放列表
POST   /api/playlists                # 创建播放列表（指定文件夹路径）
GET    /api/playlists/:id            # 播放列表详情（含排序后的媒体列表）
PUT    /api/playlists/:id            # 更新（播放模式）
DELETE /api/playlists/:id            # 删除播放列表
POST   /api/playlists/play-now       # 一键播放文件夹

# Upload
POST   /api/upload                   # 上传文件到指定目录（PC端）

# Admin
GET    /api/admin/invite-codes       # 注册码管理
POST   /api/admin/invite-codes       # 生成注册码
DELETE /api/admin/invite-codes/:code
GET    /api/admin/scan-folders       # 扫描文件夹管理
POST   /api/admin/scan-folders       # 添加扫描文件夹
DELETE /api/admin/scan-folders/:id
GET    /api/admin/config             # 获取当前配置
```

## Client Architecture

```
client/
├── public/
│   └── favicon.svg
├── src/
│   ├── components/          # 共享 UI 组件
│   │   ├── Layout.tsx       # 响应式布局壳
│   │   └── ProtectedRoute.tsx
│   ├── pages/               # 路由页面
│   │   ├── Browse.tsx       # 媒体浏览（网格/列表）
│   │   ├── MediaDetail.tsx  # 媒体详情+编辑
│   │   ├── Player.tsx       # 播放器壳（视频/音频）
│   │   ├── Playlists.tsx    # 播放列表管理
│   │   ├── Search.tsx       # 搜索页面
│   │   ├── Upload.tsx       # 文件上传（PC）
│   │   ├── Login.tsx        # 登录
│   │   ├── Register.tsx     # 注册
│   │   ├── Setup.tsx        # 管理员初始化
│   │   ├── Settings.tsx     # 设置
│   │   └── TVAuth.tsx       # TV 设备码授权
│   ├── stores/              # 状态管理 (Zustand)
│   ├── api/                 # API 调用封装 (axios + JWT interceptor)
│   ├── config.ts            # 从 import.meta.env 读取 API 地址
│   └── styles/
│       └── theme.css        # 粉蓝主题色变量
├── .env.example             # 前端配置模板（提交 git）
├── .env                     # 实际配置（gitignore）
├── vite.config.ts
└── package.json
```

## Capacitor TV Player Plugin Interface

```typescript
interface TvPlayerPlugin {
  play(options: {
    url: string
    title?: string
    coverUrl?: string
    startPosition?: number
  }): Promise<void>
  pause(): Promise<void>
  seek(options: { seconds: number }): Promise<void>
  setSpeed(options: { rate: number }): Promise<void>
  stop(): Promise<void>
  addListener(event: 'timeupdate', callback: (data: {
    currentTime: number
    duration: number
    buffered: number
  }) => void): void
  addListener(event: 'ended', callback: () => void): void
  addListener(event: 'error', callback: (data: { message: string }) => void): void
}
```
