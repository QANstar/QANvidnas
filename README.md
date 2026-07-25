# 🎬 QANvidnas

一个轻量、美观的 NAS 媒体管理与播放中心。Docker 一键部署，支持手机、电视、PC 三端访问。

## 功能特性

- 📁 **NAS 媒体库管理** — 自动扫描 NAS 文件夹，支持视频和音频格式
- 🏷️ **标签与搜索** — 自定义标签、全文搜索、组合筛选
- 📋 **播放列表** — 文件夹即播放列表，一键递归播放
- 🔀 **多种播放模式** — 顺序播放、循环播放、随机播放、单曲循环
- 🎨 **粉蓝活泼配色** — 现代 UI，响应式设计，适配手机/PC/电视
- 🔐 **用户认证** — 管理员初始化 + 注册码注册 + JWT 登录
- 📺 **TV 原生体验** — Android TV APK，ExoPlayer 硬解，遥控器交互
- 📱 **手机 APK** — 后台音频播放，通知栏播控
- 🖼️ **自动封面生成** — FFmpeg 取视频帧 + 黑场检测 + 自定义上传
- ⚡ **长视频快进优化** — 5 级渐进加速曲线，解决电视遥控器快进难题
- 📤 **文件上传** — PC 端上传音视频到 NAS

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go + Gin + SQLite (WAL + FTS5) |
| 前端 | React 18 + TypeScript + Vite + Zustand |
| 移动端 | Capacitor + ExoPlayer |
| 部署 | Docker 单容器 |
| 媒体处理 | FFmpeg (ffprobe + 缩略图生成) |

## 快速部署

### 1. 准备配置文件

```bash
cp config.example.yaml config.yaml
# 编辑 config.yaml，修改注册码等配置
```

### 2. 启动容器

```bash
docker-compose up -d
```

### 3. 访问

- 浏览器打开 `http://<NAS-IP>:3000`
- 首次使用需创建管理员账号
- 在设置页面添加 NAS 媒体文件夹路径
- 点击"手动扫描"开始索引

## 构建客户端 APK

```bash
# 安装 Capacitor 依赖
cd client && npm ci

# 构建 TV APK
npx cap sync android
cd android && ./gradlew assembleRelease

# APK 位于: client/android/app/build/outputs/apk/release/
```

## 开发

```bash
# 启动后端 (需要 Go 1.22+)
cd server && go run .

# 启动前端 (开发模式，带 API 代理)
cd client && npm run dev
```

## 支持的媒体格式

**视频:** MP4, MKV, AVI, MOV, WMV, FLV, WebM, M4V, MPG, TS  
**音频:** MP3, FLAC, AAC, OGG, WAV, M4A, WMA, OPUS

## 项目结构

```
QANvidnas/
├── server/                    # Go 后端
│   ├── internal/
│   │   ├── auth/             # JWT 认证
│   │   ├── config/           # 配置管理
│   │   ├── database/         # SQLite + 迁移
│   │   ├── handlers/         # API 处理器
│   │   ├── middleware/       # 中间件
│   │   ├── models/           # 数据模型
│   │   ├── router/           # 路由
│   │   ├── scanner/          # 媒体扫描
│   │   └── thumbnail/        # 缩略图生成
│   └── main.go
├── client/                    # React 前端 + Capacitor
│   ├── src/
│   │   ├── api/              # API 客户端
│   │   ├── components/       # 共享组件
│   │   ├── pages/            # 页面
│   │   ├── platforms/        # 平台抽象层
│   │   ├── stores/           # Zustand 状态
│   │   └── styles/           # 主题
│   └── capacitor.config.ts
├── Dockerfile
├── docker-compose.yml
├── config.example.yaml
└── Makefile
```

## License

MIT
