## 1. 项目脚手架

- [x] 1.1 初始化 .NET 项目：`server/` 目录，ASP.NET Core 9.0 minimal API，EF Core SQLite，JWT Bearer，CORS
- [x] 1.2 初始化 React 前端：`client/` 目录，Vite + React 19 + TypeScript + React Router，粉蓝主题 CSS 变量
- [ ] 1.3 配置 Makefile：`make dev`, `make build`, `make docker`, `make apk-tv`, `make apk-mobile`
- [x] 1.4 创建 `appsettings.json` + `appsettings.Example.json` 配置模板，`.gitignore` 排除 `config.yaml` 和构建产物
- [x] 1.5 创建 `client/.env.example` 前端配置模板，`.env` 加入 gitignore

## 2. 数据库与配置

- [x] 2.1 创建 EF Core DbContext 及数据模型（Users, InviteCodes, Media, Tags, MediaTags, Playlists, PlaylistItems, ScanFolders, DeviceCodes）
- [x] 2.2 实现配置加载：读取 `appsettings.json` + `config.yaml`，环境变量覆盖
- [x] 2.3 全文搜索使用 EF Core LINQ Contains 查询（替代原计划的 FTS5，消除 C 扩展依赖）
- [x] 2.4 实现数据库自动创建（EnsureCreated），后续可升级为 EF Core Migration

## 3. 用户认证系统

- [x] 3.1 实现管理员初始化 API（`POST /api/auth/setup`）：首次检测 + 创建管理员 + 关闭接口
- [x] 3.2 实现注册码注册 API（`POST /api/auth/register`）：验证注册码 → 创建用户 → 计数递增
- [x] 3.3 实现 JWT 登录 API（`POST /api/auth/login`）：用户名密码验证 → 签发 Token
- [x] 3.4 实现 JWT 认证中间件：保护所有 `/api/*` 路由（除 auth 公开接口）
- [x] 3.5 实现 TV 设备码登录：生成 4 位码 + 轮询授权 + Token 发放
- [x] 3.6 实现管理后台中间件：admin 路由额外校验 is_admin 声明

## 4. 媒体扫描引擎

- [x] 4.1 实现全量扫描：递归遍历配置路径 → 过滤支持格式 → 提取元数据（ffprobe）→ 写入 DB
- [x] 4.2 实现 ffprobe 元数据提取：时长、分辨率、编码、码率、文件大小
- [ ] 4.3 实现 FileSystemWatcher 实时监听（v2）：文件增删改事件 → 防抖 → 增量处理
- [x] 4.4 实现定时轮询降级：每 5 分钟检查新增文件
- [x] 4.5 实现手动扫描 API（`POST /api/media/scan`）：触发全量重扫 + 进度查询
- [x] 4.6 实现扫描文件夹管理 API：管理员添加/删除扫描路径

## 5. 缩略图生成

- [x] 5.1 实现封面自动生成：FFmpeg 取 10% 位置帧 → JPEG 480px → 黑场检测回退
- [x] 5.2 实现进度条雪碧图生成：每 10s 一帧 → 10 列网格排列 → JPEG 保存 + 元数据入库
- [x] 5.3 实现异步生成队列：封面和雪碧图 `Task.Run` 后台生成，不阻塞扫描主流程
- [x] 5.4 实现默认占位图：视频（播放图标）、音频（音符图标）作为 SVG 内嵌

## 6. 媒体管理 API

- [x] 6.1 实现媒体列表 API（`GET /api/media`）：分页 + 排序（名称/日期/时长）+ 类型筛选
- [x] 6.2 实现媒体详情 API（`GET /api/media/:id`）
- [x] 6.3 实现媒体元数据编辑 API（`PUT /api/media/:id`）：title, description, tags
- [x] 6.4 实现自定义封面上传 API（`POST /api/media/:id/cover`）：接收图片 → 覆盖自动封面
- [x] 6.5 实现标签 CRUD API（`GET/POST/DELETE /api/tags`）

## 7. 搜索功能

- [x] 7.1 实现搜索 API（`GET /api/search`）：EF Core LINQ Contains → 标题/描述/标签匹配
- [x] 7.2 实现标签筛选 API：单标签 / 多标签 AND 逻辑
- [x] 7.3 实现组合筛选：搜索关键词 + 标签 + 类型 三条件交集

## 8. 媒体流服务

- [x] 8.1 实现视频流 Handler（`GET /api/stream/video/:id`）：HTTP Range 支持 → 206 Partial Content
- [x] 8.2 实现音频流 Handler（`GET /api/stream/audio/:id`）：HTTP Range 支持
- [x] 8.3 实现封面图服务（`GET /api/stream/cover/:id`）：返回封面文件 + 雪碧图
- [x] 8.4 实现路径安全校验：防止路径遍历攻击
- [x] 8.5 实现 MIME 类型自动识别（根据文件扩展名设置 Content-Type）

## 9. 播放列表管理

- [x] 9.1 实现播放列表 CRUD API：创建（指定文件夹路径 → 递归扫描子文件夹）→ 查看 → 删除
- [x] 9.2 实现播放模式切换 API（`PUT /api/playlists/:id`）：sequential / loop / random / single-loop
- [x] 9.3 实现 Fisher-Yates 洗牌算法：随机播放模式下打乱列表
- [x] 9.4 实现一键播放 API：创建临时播放列表 + 返回第一项播放信息

## 10. 文件上传

- [x] 10.1 实现文件上传 API（`POST /api/upload`）：Multipart 接收 → 流式写入 → 支持 10GB
- [x] 10.2 实现上传限制：格式校验 + 大小上限 10GB + 目标目录校验
- [ ] 10.3 实现上传后自动触发扫描：文件写入完成后加入扫描队列

## 11. 前端：核心框架

- [x] 11.1 配置 Vite + React Router，响应式布局（手机/平板/PC 三档断点）
- [x] 11.2 实现粉蓝主题色系统：CSS 自定义属性，深色背景 + 粉蓝高亮
- [x] 11.3 实现 API 客户端封装：axios + JWT token interceptor + 401 自动跳转登录
- [x] 11.4 实现状态管理（Zustand）：用户状态、播放器状态、过滤条件
- [ ] 11.5 实现平台抽象层：`platforms/web/` 和 `platforms/tv/` 播放器接口定义
- [x] 11.6 实现 `.env` 前端配置：`VITE_API_BASE_URL` 构建时注入

## 12. 前端：认证页面

- [x] 12.1 管理员初始化页面：检测系统状态 → 创建首个管理员账号
- [x] 12.2 登录页面：用户名 + 密码 → 获取 JWT → 存储 token
- [x] 12.3 注册页面：注册码 + 用户名 + 密码 → 注册
- [x] 12.4 TV 设备码授权页面：输入 4 位码 → 确认授权

## 13. 前端：媒体浏览与搜索

- [x] 13.1 媒体网格浏览页：响应式网格布局 + 封面卡片 + 标签徽标 + 类型筛选
- [x] 13.2 文件夹树浏览：文件夹结构展示 + 每个文件夹的"播放"和"添加到播放列表"按钮
- [x] 13.3 搜索栏：关键词搜索 + 标签多选筛选 + 搜索结果高亮
- [x] 13.4 媒体详情页：封面大图 + 元数据显示 + 标签编辑 + 播放按钮

## 14. 前端：媒体编辑

- [x] 14.1 编辑表单：修改名称、描述
- [x] 14.2 标签编辑器：添加/移除标签，创建新标签（带颜色选择器）
- [x] 14.3 封面上传组件：拖拽上传 + 预览裁剪 + 删除恢复自动封面

## 15. 前端：Web 播放器

- [x] 15.1 视频播放器壳：HTML5 `<video>` 标签 → 自定义控制栏（播放/暂停/进度条/音量/全屏）
- [x] 15.2 音频播放器壳：`<audio>` 标签 → 迷你播放栏 + 全屏播放页
- [x] 15.3 进度条雪碧图预览：悬停进度条 → 显示对应时间缩略图
- [x] 15.4 播放列表侧栏：当前播放列表显示 + 点击切换 + 拖拽排序
- [x] 15.5 播放模式切换按钮：顺序/循环/随机/单曲循环，图标切换

## 16. 前端：PC 上传页

- [x] 16.1 上传页面：拖拽区域 + 文件选择 + 目标目录选择器
- [x] 16.2 上传进度展示：百分比 + 速度 + 剩余时间 + 队列管理
- [x] 16.3 上传完成提示：自动跳转到文件所在浏览页

## 17. 前端：设置页

- [x] 17.1 管理员设置页：扫描文件夹管理（添加/删除）+ 手动扫描按钮 + 扫描状态
- [x] 17.2 注册码管理：生成注册码 + 查看使用情况 + 禁用
- [x] 17.3 个人设置：修改密码

## 18. Capacitor 工程与原生播放器

- [ ] 18.1 初始化 Capacitor 工程：`npx cap init` + Android 平台添加
- [ ] 18.2 实现 TvPlayer Capacitor Plugin 接口定义（TypeScript 侧）
- [ ] 18.3 实现 TvPlayerPlugin.kt（Android 原生侧）：ExoPlayer 初始化 + SurfaceView 层级管理
- [x] 18.4 实现 JSBridge 通信：播放/暂停/seek/倍速 命令 + timeupdate/ended/error 事件回调
- [x] 18.5 实现 MediaSession 集成：手机通知栏播控 + 锁屏信息（封面、标题）

## 19. Android TV 定制

- [ ] 19.1 配置 Android TV Manifest：leanback intent filter + D-pad 支持 + 桌面图标
- [ ] 19.2 实现 TV 端焦点导航系统：方向键焦点移动 + 焦点框动画 + 返回键栈管理
- [ ] 19.3 实现长按加速快进：5 级加速曲线 → ExoPlayer seek → UI 倍率指示器
- [ ] 19.4 实现 TV 端播放器页面：遥控器媒体键映射 + OK 键取消快进 + 进度条缩略图预览
- [ ] 19.5 实现 TV 设备码登录页：显示设备码 → 轮询授权状态 → 自动跳转

## 20. 手机 APK 定制

- [ ] 20.1 配置 Capacitor Android 移动端：手机样式适配 + 触摸交互
- [ ] 20.2 实现手机端后台音频播放：Service + MediaSession → 切后台/锁屏继续播放
- [ ] 20.3 实现手机端播放器：触摸进度条 + 双击快进/快退 + 手势音量/亮度调节

## 21. Docker 部署

- [x] 21.1 编写 Dockerfile.backend：多阶段构建（.NET SDK → aspnet runtime + FFmpeg）
- [x] 21.2 编写 Dockerfile.frontend：两阶段构建（Node → React build → Nginx 静态服务）
- [x] 21.3 编写 nginx.conf：SPA fallback + /api 反向代理到 backend
- [x] 21.4 编写 docker-compose.yml：双容器编排 + 网络 + volume 挂载
- [ ] 21.5 构建 Docker 镜像并验证在飞牛 NAS 上部署运行
- [x] 21.6 编写 docker-compose.example.yml 部署模板

## 22. 构建与发布

- [ ] 22.1 配置 APK 签名：生成 keystore + 签名配置
- [ ] 22.2 编写 APK 构建脚本：`make apk-tv` → 构建 React → cap sync → gradle build → 输出 APK
- [ ] 22.3 编写手机 APK 构建脚本：`make apk-mobile`
- [ ] 22.4 APK Release 页面（GitHub Releases）：上传 TV APK + 手机 APK + 更新日志

## 23. 收尾

- [x] 23.1 README 更新：项目介绍 + 功能列表 + 部署指南 + 截图
- [ ] 23.2 `.claude/settings.json` 项目配置（如需要）
- [ ] 23.3 端到端测试：Docker 部署 → 初始化 → 扫描 → 浏览 → 播放 → TV 登录 → TV 播放
