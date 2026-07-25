# config-management

配置管理。YAML 配置文件管理（注册码、扫描文件夹等），样例配置提交 Git，真实配置 Git 忽略。

## ADDED Requirements

### Requirement: YAML 配置文件

系统 SHALL 从 `config.yaml` 文件读取配置。若文件不存在，系统使用内置默认值启动并生成示例配置。配置变更后部分选项支持热加载。

#### Scenario: 配置文件存在

- **WHEN** 系统启动时 `config.yaml` 存在且格式正确
- **THEN** 系统加载配置并应用到运行中

#### Scenario: 配置文件不存在

- **WHEN** 系统首次启动且 `config.yaml` 不存在
- **THEN** 系统以默认配置启动（无预置注册码、无扫描文件夹），需管理员初始化设置

#### Scenario: 配置文件格式错误

- **WHEN** `config.yaml` 存在但 YAML 格式错误
- **THEN** 系统启动失败并输出明确的错误信息，指出语法错误位置

### Requirement: 样例配置文件提交 Git

项目仓库 SHALL 包含 `config.example.yaml` 样例文件，展示所有配置项及其注释说明。`config.yaml` 在 `.gitignore` 中排除。

#### Scenario: 新用户参考样例配置

- **WHEN** 用户首次部署，查看仓库中的 `config.example.yaml`
- **THEN** 用户能根据注释理解每个配置项的含义并复制为 `config.yaml`

#### Scenario: config.yaml 不被提交

- **WHEN** 开发者执行 `git add` 或 `git commit`
- **THEN** `.gitignore` 确保 `config.yaml` 不被纳入版本控制

### Requirement: 注册码配置

注册码 SHALL 在配置文件中定义，每项包含：邀请码字符串、描述、最大使用次数（0 = 无限）、过期时间（空 = 永不过期）。

#### Scenario: 配置文件定义注册码

- **WHEN** 配置文件中包含注册码定义
- **THEN** 系统加载注册码供用户注册使用

#### Scenario: 管理员在应用内管理注册码

- **WHEN** 管理员通过应用 API 创建/删除注册码
- **THEN** 系统同时更新 `config.yaml` 文件以保持持久化

### Requirement: 配置热更新

系统 SHALL 监听 `config.yaml` 的文件变更，当检测到修改时自动重新加载配置，无需重启 Docker 容器。

#### Scenario: 手动修改配置文件

- **WHEN** 管理员在 NAS 上编辑 `config.yaml` 并保存
- **THEN** 系统检测到文件变更，重新加载配置，新注册码等即时生效

### Requirement: 敏感信息脱敏

系统 API 返回配置信息时 SHALL 对敏感字段（如注册码的使用情况）进行脱敏处理。`/api/admin/config` 仅管理员可访问。

#### Scenario: 管理员查看配置

- **WHEN** 管理员访问 `/api/admin/config`
- **THEN** 系统返回当前配置，注册码显示完整信息

#### Scenario: 非管理员无法查看配置

- **WHEN** 非管理员用户访问 `/api/admin/config`
- **THEN** 系统返回 403
