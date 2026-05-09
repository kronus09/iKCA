# iKCA - IKEv2 证书生成器

基于 Go 语言开发的 IKEv2 自签证书生成工具，支持 50 年有效期证书，提供 Web UI 和 CLI 两种使用方式。

## 功能特性

- ✅ **Web UI 交互界面**：可视化填写参数，一键生成
- ✅ **CLI 命令行工具**：脚本化批量生成
- ✅ **50 年证书有效期**：告别频繁换证
- ✅ **支持 Windows/Android/iOS 原生连接**：无需额外客户端
- ✅ **自动持久化存储**：生成证书自动保存到 `data/` 目录
- ✅ **自动加载已有证书**：重启后自动加载，无需重新生成
- ✅ **Docker 一键部署**：支持容器化运行
- ✅ **证书下载**：直接下载 p12/pem/crt 格式证书
- ✅ **清理功能**：一键清理已生成的证书

## 安装使用

### 方式一：直接运行（Go 环境）
```bash
# 克隆项目
git clone <your-repo>
cd ikca

# 编译运行（推荐）
go build -o ikca .
./ikca -mode web

# 或快速运行（不编译，直接启动）
go run .

# 或 CLI 模式生成证书
go run . -mode cli -domain your.domain.com -ca-pass 123456 -client-pass 123456
```

### 方式二：Docker 运行
```bash
# 使用 Docker Compose
docker-compose up -d

# 或直接运行
docker run -d -p 20509:20509 -v ./data:/app/data --name ikca ikca:latest
```

启动后访问：`http://localhost:20509`

## 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-mode` | `web` | 运行模式：`web`（Web UI）或 `cli`（命令行） |
| `-listen` | `:20509` | Web 服务监听地址 |
| `-data-dir` | `./data` | 证书保存目录（Docker 容器内为 `/app/data`） |
| `-domain` | `""` | 服务器域名（CLI 模式必填） |
| `-clients` | `vpnclient` | 客户端名称，空格分隔多个 |
| `-ca-pass` | `""` | CA 证书密码（可通过 CA_PASS 环境变量设置） |
| `-client-pass` | `""` | 客户端证书密码（可通过 CLIENT_PASS 环境变量设置） |

## 证书文件说明

生成的文件位于 `data/` 目录：

```
data/
├── ca.p12              # CA 证书（含私钥），导入系统时使用
├── caCert.pem          # CA 证书（PEM 格式）
├── caCert.crt          # CA 证书（DER 格式）
├── serverCert_[domain].pem   # 服务端证书
├── serverKey_[domain].pem    # 服务端私钥
├── client_[name].p12         # 客户端证书（含私钥），导入设备使用
└── clientCert_[name].crt     # 客户端证书（DER 格式）
```

## iKuai 路由器配置

1. 在 Web UI 填写参数生成证书
2. 下载 `serverCert_*.pem` 和 `serverKey_*.pem`
3. 进入 iKuai 路由器管理界面：
   - 路径：VPN → IKEv2/IPSec 服务端
   - 类型：IKEv2/IPSec MSCHAPv2
   - 本地标识：填写你的域名
   - 服务端证书：粘贴 `serverCert_*.pem` 内容
   - 私钥：粘贴 `serverKey_*.pem` 内容
   - 开启服务端状态 → 保存

4. 创建 VPN 账号：
   - 路径：认证计费 → 认证账号管理 → 账号管理
   - 添加账号，设置用户名/密码
   - 认证类型可选择限定 ikev2 或不限

## 客户端配置

### 第一步：安装 CA 证书（所有客户端都需要）

**Windows**:
1. 双击 `caCert.crt` 或 `ca.p12`
2. 选择"将所有的证书都放入下列存储" → "浏览" → "受信任的根证书颁发机构" → "确定"
3. 完成安装

**Android**:
1. 将 `caCert.crt` 复制到手机
2. 设置 → 安全 → 加密与凭据 → 从存储设备安装证书 → CA证书
3. 选择证书文件，命名后安装

**iOS/macOS**:
1. 通过邮件或 AirDrop 发送 `caCert.crt` 到设备
2. 打开文件，按提示安装到"证书"中
3. 进入设置 → 通用 → 关于本机 → 证书信任设置
4. 启用对根证书的完全信任

### 第二步：安装客户端证书和设备连接

#### Windows 10/11
1. 双击 `client_*.p12` 导入到"本地计算机"
2. 新建 VPN 连接：
   - 类型：IKEv2
   - 服务器地址：你的域名
   - 用户名/密码：iKuai 中设置的账号
   - 连接

#### Android
1. 将 `client_*.p12` 复制到手机
2. 设置 → 安全 → 加密与凭据 → 安装证书
3. 选择 VPN 证书，安装时需要输入密码
4. 新建 VPN 连接：
   - 类型：IKEv2/IPSec
   - 服务器地址：你的域名
   - 用户名/密码：iKuai 中设置的账号
   - 连接

#### iOS/macOS
1. 通过 AirDrop、邮件等方式发送 `client_*.p12` 到设备
2. 点击安装，输入密码（即 `-client-pass` 设置的密码）
3. 系统会提示安装完成
4. 设置 → VPN → 添加 VPN 配置 → IKEv2
   - 描述：自定义名称
   - 服务器：你的域名
   - 远程ID：留空或填域名
   - 本地ID：留空
   - 用户名/密码：iKuai 中设置的账号
   - 使用证书：选择刚才安装的客户端证书

## Docker 部署

### docker-compose.yml
```yaml
version: '3.8'
services:
  ikca:
    build: .
    image: ikca:latest
    container_name: ikca
    ports:
      - "20509:20509"
    volumes:
      - ./data:/app/data
    environment:
      - TZ=Asia/Shanghai
    restart: unless-stopped
```

### 直接运行
```bash
# 构建镜像
docker build -t ikca:latest .

# 运行容器
docker run -d \
  --name ikca \
  -p 20509:20509 \
  -v ./data:/app/data \
  ikca:latest
```

## 环境变量

| 变量名 | 说明 |
|--------|------|
| `CA_PASS` | CA 证书密码（CLI 模式） |
| `CLIENT_PASS` | 客户端证书密码（CLI 模式） |
| `TZ` | 时区设置，建议 `Asia/Shanghai` |

## API 接口

Web 服务提供以下 REST API：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/generate` | POST | 生成证书，JSON 参数 |
| `/api/download/ca` | GET | 下载 CA p12 文件 |
| `/api/download/ca-cert` | GET | 下载 CA 证书（PEM） |
| `/api/download/ca-crt` | GET | 下载 CA 证书（DER/CRT） |
| `/api/download/server-cert` | GET | 下载服务端证书 |
| `/api/download/server-key` | GET | 下载服务端私钥 |
| `/api/download/client?name=xxx` | GET | 下载客户端 p12 文件 |
| `/api/list-data` | GET | 列出已生成的证书文件 |
| `/api/clear` | POST | 清理所有已生成的证书 |

## 技术栈

- **后端**：Go 语言 + 标准库 `net/http`
- **证书生成**：`crypto/x509` + `software.sslmate.com/src/go-pkcs12`
- **前端**：原生 HTML + JavaScript + Tailwind CSS CDN
- **部署**：Docker + Docker Compose
- **数据持久化**：本地文件系统（`/app/data` 目录）

## 注意事项

1. **密码安全**：CA 密码和客户端密码请妥善保管
2. **域名匹配**：服务端域名必须与证书域名完全一致
3. **客户端名称**：仅用于区分不同设备证书，不影响 IKEv2 登录用户名
4. **iOS/macOS**：确保共享 SAN 字段设置为 `IKEv2Clients` 或自定义值
5. **证书有效期**：默认 50 年，CA 默认 10 年，可通过参数调整

## 故障排除

### "IKE 身份验证凭证不可接受"
- 检查服务端域名是否与证书域名匹配
- 确认客户端已正确安装 CA 证书或 p12 文件
- 确保 iKuai 配置的本地标识与域名一致

### 客户端无法连接
- 检查防火墙是否放行 500/4500 端口
- 确认路由器已开启 IKEv2 服务端状态
- 检查 VPN 账号密码是否正确

### 证书导入失败
- 确认输入的密码正确（CLI 模式的 `-client-pass` 参数）
- Windows 导入时选择"本地计算机"而不是"当前用户"

## 许可证

MIT License
