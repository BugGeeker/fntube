# AGENT.md

## 项目概述

FnTube 是一个飞牛（fnOS）应用包。应用通过飞牛应用中心进行安装、升级、卸载和配置，遵循飞牛应用包规范。

- **应用名称**: FnTube
- **版本**: 1.0.0
- **平台**: x86
- **来源**: thirdparty
- **维护者**: BugGeeker

## 技术栈

### 后端
- **语言**: Go
- **Web 框架**: CloudWeGo Hertz
- **数据库**: SQLite

### 前端
- **框架**: Vue 3
- **语言**: TypeScript
- **UI 组件库**: Ant Design Vue (antdv)
- **状态管理**: Pinia

## 目录结构

```
fntube/
├── manifest                 # 应用清单文件，定义应用元信息
├── ICON.png                 # 应用图标（大）
├── ICON_256.png             # 应用图标（256px）
├── app/
│   └── ui/
│       ├── config           # UI 配置，定义桌面入口（协议、端口、URL 路径等）
│       └── images/
│           ├── icon_256.png # 桌面图标 256px
│           └── icon_64.png  # 桌面图标 64px
├── cmd/                     # 应用生命周期脚本
│   ├── main                 # 主进程管理脚本（start/stop/status）
│   ├── install_init         # 安装前钩子
│   ├── install_callback     # 安装后钩子
│   ├── uninstall_init       # 卸载前钩子
│   ├── uninstall_callback   # 卸载后钩子
│   ├── upgrade_init         # 升级前钩子
│   ├── upgrade_callback     # 升级后钩子
│   ├── config_init          # 配置变更前钩子
│   └── config_callback      # 配置变更后钩子
└── config/
    ├── privilege            # 权限配置（运行身份等）
    └── resource             # 资源配置（共享目录等）
```

## 飞牛应用包规范

### manifest 清单文件

定义应用的基本信息，包括应用名、版本、显示名、描述、平台、来源、维护者、分发者、桌面 UI 目录及应用启动名等关键字段。

### 生命周期脚本（cmd/）

所有脚本为 bash 脚本，在应用生命周期的不同阶段被飞牛系统调用：

| 脚本 | 触发时机 |
|------|----------|
| `main` | 应用启动/停止/状态查询（支持 `start`、`stop`、`status` 参数） |
| `install_init` | 安装前 |
| `install_callback` | 安装后 |
| `uninstall_init` | 卸载前 |
| `uninstall_callback` | 卸载后 |
| `upgrade_init` | 升级前 |
| `upgrade_callback` | 升级后 |
| `config_init` | 环境变量变更前 |
| `config_callback` | 环境变量变更后 |

### main 脚本说明

`main` 脚本负责管理后端进程的启动、停止和状态检查：

- 使用 `${TRIM_PKGVAR}` 环境变量获取应用数据目录
- 日志写入 `${TRIM_PKGVAR}/info.log`
- PID 记录在 `${TRIM_PKGVAR}/app.pid`
- 支持 `start`（启动）、`stop`（停止，先 TERM 后 KILL）、`status`（状态检查）三个命令
- 需要在 `CMD` 变量中填写实际的启动命令

### 权限配置（config/privilege）

默认以 `package` 身份运行。

### 资源配置（config/resource）

定义数据共享目录（`fntube`、`fntube/data`）。

### UI 配置（app/ui/config）

定义桌面入口为 URL 类型，通过 HTTP 协议访问后端服务，端口和路径使用 `{port}` 和 `{url-path}` 占位符。

## 开发约定

### 后端

- 使用 Go 编写后端服务，通过 Hertz 框架提供 HTTP API
- 数据持久化使用 SQLite
- 编译产物需部署为飞牛应用包内的可执行文件，由 `cmd/main` 脚本管理启停
- API 需与前端对接，URL 路径需与 `app/ui/config` 中的 `url-path` 一致

### 前端

- 使用 Vue 3 + TypeScript 开发，UI 基于 Ant Design Vue
- 状态管理使用 Pinia
- 构建产物部署至 `app/www/` 目录下，由后端静态托管
- 桌面图标使用 `app/ui/images/` 下的 `icon_256.png` 和 `icon_64.png`

### 通用

- 修改应用元信息时同步更新 `manifest` 文件
- 新增共享目录或调整权限时修改 `config/` 下对应配置
- 生命周期脚本修改后确保脚本具有可执行权限
- 遵循飞牛应用包目录规范，不随意变更固定路径下的文件结构

### 飞牛应用打包规范

- 后端服务需编译为可执行文件，部署至 `app/server/` 目录下
- 前端构建产物需部署至 `app/www/` 目录下
- 打包fpk必须使用fnpack命令，禁止直接使用tar打包

### 本地调试

- 使用 `dev-start.bat` 启动本地调试环境
- 使用 `dev-stop.bat` 停止本地调试环境