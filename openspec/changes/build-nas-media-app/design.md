## Context

QANvidnas 是一个自托管的 NAS 媒体管理与播放应用。部署在飞牛 NAS (x86) 的 Docker 环境，为家庭用户提供手机/电视/PC 三端统一的音视频浏览、搜索和播放体验。项目从零构建，无历史代码负担。

**约束条件：**
- 必须 Docker 单容器部署，降低用户操作门槛
- 服务端 Go，前端 React（团队技术偏好）
- Android TV 体验要求原生级别（硬解、遥控器交互）
- 手机端需支持后台音频播放
- HTTP 局域网通信（v1 不要求 HTTPS）
- 所有媒体访问需要登录认证

## Goals / Non-Goals

**Goals:**
- Go 单体服务，内嵌 React SPA，一个二进制 + 一个 Dockerfile 完成部署
- Capacitor 包装 Android APK（手机+TV），WebView 加载远程 UI，原生 ExoPlayer 处理播放
- 注册码用户体系，管理员控制谁可以访问
- 文件夹级媒体库管理，标签+全文搜索
- 电视端渐进加速快进，解决长视频遥控器操作痛点
- 自动封面生成（FFmpeg）+ 进度条雪碧图预览

**Non-Goals:**
- 不做转码/实时转码（直接流传输原文件，客户端硬解）
- 不做字幕支持
- v1 不做 HTTPS（后续用 Caddy sidecar 或用户自配反向代理）
- 不做 DLNA/UPnP 协议支持
- 不做多语言国际化（仅中文）
- 不做 iOS 客户端（v1 仅 Android）
- 不做 ISO 原盘/蓝光结构解析

## Decisions

### D1: Go 单体 + 内嵌前端 vs 前后端分离部署

**选择：Go 单体，`embed.FS` 内嵌 React 构建产物。**

理由：目标用户是 NAS 家庭用户，Docker 单容器部署是硬需求。分离部署需要用户管理多个容器或挂载前端文件，显著增加部署失败率。Go 1.16+ 的 embed 天然支持，且编译后二进制仅增大 ~5MB（gzip 后前端产物）。

替代方案：前后端分离（Nginx + Go 双容器，或用户手动挂载 web/dist）。被拒绝——增加部署复杂度，对家庭用户不友好。

### D2: SQLite vs PostgreSQL

**选择：SQLite (WAL 模式 + FTS5 全文搜索)。**

理由：家庭场景并发 < 10，SQLite 完全胜任。WAL 模式下读并发不受限。FTS5 提供内置全文搜索能力，无需额外部署 Elasticsearch/Meilisearch。零维护——不需要单独数据库容器，数据文件随应用持久化。

替代方案：PostgreSQL——更强并发但需要独立容器，对家庭场景过度设计。

### D3: Capacitor vs React Native vs 纯 WebView

**选择：Capacitor + 自定义 ExoPlayer 插件。**

理由：
- Capacitor 是标准化的 WebView 壳方案，不改变 React 代码结构
- 自定义 Plugin 机制让原生 ExoPlayer 与 JS 层通过类型化接口通信
- 手机和 TV 共享同一个 Capacitor 工程，仅平台配置不同
- React 代码 95%+ 复用，平台差异仅限播放器桥接层

替代方案：React Native——需重写全部 UI 层，TV 支持不成熟。纯 WebView——JSBridge 需手写协议，不如 Capacitor Plugin API 规范。

### D4: TV 端 ExoPlayer 原生播放 vs HLS/DASH 自适应流

**选择：ExoPlayer 原生播放，Go 后端直接 serve 原始文件 + HTTP Range。**

理由：NAS 局域网带宽充足（通常 1Gbps），无需自适应码率。ExoPlayer 直接解码 MKV/MP4/AVI 等原始格式，零转码开销。Go Stream Handler 仅处理 Range 请求实现 seek。

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

### D7: 文件监听：fsnotify + 定时轮询降级

**选择：fsnotify 实时监听 IN_CREATE/IN_MOVED_TO 事件（防抖 5s），同时每 5 分钟执行一次 mtime 快速检查作为降级方案。提供"重新扫描"手动触发按钮。**

理由：Docker 容器对宿主机挂载卷的 inotify 支持取决于文件系统和挂载方式，某些 NAS 配置下不可靠。轮询降级保证任何场景下都能检测到新文件。

### D8: 随机播放：Fisher-Yates 洗牌

**选择：播放列表加载时执行 Fisher-Yates 洗牌，生成随机排列后顺序播放。切换到"下一个"沿洗牌后列表前进，保证不重复。**

理由：避免"每次随机取一首"可能导致的短期重复问题。用户切完所有歌之前每首只播一次。

### D9: Monorepo 单仓库

**选择：`server/` (Go) + `client/` (React+Capacitor) 同仓库，Makefile 统一构建入口。**

理由：React 源码是三端的唯一 UI 来源；Server 内嵌 React 产物；Capacitor APK 加载同一份 React 源码构建的远程 UI。拆分仓库会增加构建产物传递的复杂度（需 CI artifact pipeline 或 npm 私有包），对当前项目规模是过度工程化。

## Risks / Trade-offs

- **[R1] Docker 挂载卷的 fsnotify 可能不工作** → 降级到 5 分钟轮询，确保功能不缺失。同时在管理面板显示"上次扫描时间"让用户感知状态。
- **[R2] Android TV WebView 性能差异大** → 播放器走原生 ExoPlayer，WebView 仅负责 UI。UI 端避免复杂动画，使用 CSS transform 而非 JS 动画。低端设备降低雪碧图帧数。
- **[R3] FFmpeg 依赖** → Docker 镜像需包含 ffmpeg（约 +30MB）。在 Dockerfile 中固定 FFmpeg 版本。封面/缩略图生成异步执行，不阻塞扫描主流程。
- **[R4] SQLite 大库性能** → 10 万级文件下 FTS5 搜索依然 < 100ms。如未来遇到瓶颈可迁移 PostgreSQL（但大概率不需要）。
- **[R5] Capacitor Plugin 原生 ExoPlayer 与 WebView 的 SurfaceView 层级** → SurfaceView 默认在最上层，可能遮挡 WebView 弹窗。播放控制 UI 由 React 渲染在 WebView 中，需调整布局避免被遮挡区域。

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    飞牛 NAS (Docker)                              │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    QANvidnas 容器                          │  │
│  │                                                           │  │
│  │   Go Server (:3000)                                       │  │
│  │   ┌───────────────────────────────────────────────────┐  │  │
│  │   │  embed.FS ─── React SPA (web/dist/)               │  │  │
│  │   └───────────────────────────────────────────────────┘  │  │
│  │                                                           │  │
│  │   ┌──────────┐ ┌──────────┐ ┌──────────────────────┐    │  │
│  │   │  Auth    │ │  Media   │ │  Stream              │    │  │
│  │   │  /api/   │ │  /api/   │ │  /api/stream/        │    │  │
│  │   │  auth/*  │ │  media/* │ │  (Range handler)     │    │  │
│  │   └──────────┘ └──────────┘ └──────────────────────┘    │  │
│  │                                                           │  │
│  │   ┌──────────┐ ┌──────────┐ ┌──────────────────────┐    │  │
│  │   │ Playlist │ │  Search  │ │  Thumbnail           │    │  │
│  │   │ /api/    │ │  /api/   │ │  (FFmpeg worker)     │    │  │
│  │   │ playlist │ │  search  │ │                      │    │  │
│  │   └──────────┘ └──────────┘ └──────────────────────┘    │  │
│  │                                                           │  │
│  │   ┌──────────┐ ┌──────────────────────────────────────┐  │  │
│  │   │  Watch   │ │  SQLite (WAL + FTS5)                 │  │  │
│  │   │ fsnotify │ │  /data/qanvidnas.db                  │  │  │
│  │   │ + poll   │ └──────────────────────────────────────┘  │  │
│  │   └──────────┘                                           │  │
│  └───────────────────────────────────────────────────────────┘  │
│        │                  │                                    │
│   ┌────┴────┐        ┌────┴────┐                              │
│   │ /config │        │ /media  │  ← Docker volume mounts      │
│   │ .yaml   │        │ (NAS)   │                              │
│   └─────────┘        └─────────┘                              │
└─────────────────────────────────────────────────────────────────┘
```

## Data Model

```
User: id, username, password_hash, is_admin, created_at
InviteCode: code, description, max_uses, used, expires_at
Media: id, title, description, type(video/audio), path, duration,
       resolution, cover_path, sprite_path, file_size, codec,
       bitrate, created_at, updated_at
Tag: id, name, color
MediaTag: media_id, tag_id (M:N)
Playlist: id, name, folder_path, user_id, play_mode, created_at
PlaylistItem: playlist_id, media_id, position
ScanFolder: id, path, last_scan_at, status
```

## API Route Design

```
# Auth
POST   /api/auth/setup              # 管理员初始化（仅首次）
POST   /api/auth/login              # 登录 → JWT
POST   /api/auth/register           # 注册（需注册码）
GET    /api/auth/me                 # 当前用户信息
POST   /api/auth/device-code        # 生成 TV 设备码

# Media
GET    /api/media                   # 媒体列表（分页、筛选、排序）
GET    /api/media/:id               # 媒体详情
PUT    /api/media/:id               # 编辑元数据（名称、描述、标签）
POST   /api/media/:id/cover         # 上传自定义封面
GET    /api/media/:id/sprite        # 获取雪碧图元数据
POST   /api/media/scan              # 手动触发扫描

# Stream
GET    /api/stream/video/:id        # 视频流（Range 支持）
GET    /api/stream/audio/:id        # 音频流（Range 支持）
GET    /api/stream/cover/:id        # 封面图

# Search
GET    /api/search?q=&tags=&type=   # 全文搜索 + 标签筛选

# Tags
GET    /api/tags                    # 所有标签列表
POST   /api/tags                    # 创建标签
DELETE /api/tags/:id                # 删除标签

# Playlist
GET    /api/playlists               # 用户播放列表
POST   /api/playlists               # 创建播放列表（指定文件夹路径）
GET    /api/playlists/:id           # 播放列表详情（含排序后的媒体列表）
PUT    /api/playlists/:id           # 更新（播放模式）
DELETE /api/playlists/:id           # 删除播放列表

# Upload
POST   /api/upload                  # 上传文件到指定目录（PC端）

# Admin
GET    /api/admin/invite-codes      # 注册码管理
POST   /api/admin/invite-codes      # 生成注册码
DELETE /api/admin/invite-codes/:code
GET    /api/admin/scan-folders      # 扫描文件夹管理
POST   /api/admin/scan-folders      # 添加扫描文件夹
DELETE /api/admin/scan-folders/:id
GET    /api/admin/config            # 获取当前配置（脱敏）
```

## Client Architecture

```
client/
├── src/
│   ├── components/          # 共享 UI 组件
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
│   │   └── Settings.tsx     # 设置
│   ├── hooks/               # 自定义 hooks
│   ├── stores/              # 状态管理 (Zustand)
│   ├── api/                 # API 调用封装
│   ├── platforms/           # 平台抽象层
│   │   ├── web/
│   │   │   └── player.ts    # 浏览器 <video> 播放器
│   │   └── tv/
│   │       └── player.ts    # Capacitor TvPlayer 插件封装
│   └── styles/
│       ├── theme.css        # 粉蓝主题色变量
│       ├── tv.css           # TV 端适配样式
│       └── mobile.css       # 移动端适配样式
├── capacitor.config.ts
├── android/                 # Capacitor 生成的 Android 工程
├── plugins/
│   └── tv-player/           # 自定义 Capacitor Plugin
│       ├── src/
│       │   └── definitions.ts  # Plugin 接口定义
│       └── android/
│           └── TvPlayerPlugin.kt  # ExoPlayer 的原生实现
└── package.json
```

## Capacitor TV Player Plugin Interface

```typescript
// Plugin 接口定义
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

## Open Questions

- Docker 基础镜像选 Alpine 还是 Debian-slim？Alpine 更小但需确认 FFmpeg 在 musl 上的兼容性。
- 电视端是否需要画中画 (PIP)？ExoPlayer 支持，但 v1 优先级待定。
- 是否需要播放历史/续播功能？数据模型已有基础支持，可后续迭代。
