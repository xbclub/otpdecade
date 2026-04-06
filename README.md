# otpdecade

从 Google Authenticator 导出的二维码中批量提取 OTP 密钥，输出为 CSV 格式。

## 安装

```bash
go install otpdecade@latest
```

或从源码构建：

```bash
git clone https://github.com/xbclub/otpdecade.git
cd otpdecade
go build -o otpdecade .
```

## 使用方法

### 基本用法

处理单个二维码图片，结果输出到终端：

```bash
otpdecade qr.png
```

### 输出到文件

```bash
otpdecade -o secrets.csv qr.png
```

### 批量处理多个文件

Google Authenticator 导出多个账户时会生成多个二维码，可以一次传入：

```bash
otpdecade -o secrets.csv qr1.png qr2.png qr3.png
```

### 扫描目录

自动扫描目录下所有图片文件（PNG/JPG/JPEG）：

```bash
otpdecade -dir -o secrets.csv ./screenshots/
```

### 追加模式

向已有的 CSV 文件追加新提取的密钥（自动去重）：

```bash
otpdecade -append -o secrets.csv new_qr.png
```

## 从手机获取二维码

1. 打开 Google Authenticator
2. 点击右上角 **⋮** → **传输账户** → **导出账户**
3. 勾选要导出的账户，点击 **下一步**
4. 屏幕会显示二维码，**截图**保存
5. 如果账户较多，会生成多个二维码，逐一截图
6. 将截图传到电脑，用 otpdecade 处理

> **注意**：截图需要包含完整的二维码，确保清晰不模糊。

### HEIC 格式

iPhone 截图默认为 HEIC 格式，需要先转为 PNG：

```bash
# macOS 自带工具
sips -s format png photo.HEIC --out photo.png
```

## 参数说明

| 参数 | 说明 |
|------|------|
| `-o <file>` | 输出文件路径（默认输出到终端） |
| `-append` | 追加到已有文件（需配合 `-o` 使用） |
| `-dir` | 目录扫描模式，扫描所有图片文件 |

## 输出格式

CSV 文件包含两列：

```csv
account,secret
GitHub:username,JBSWY3DPEHPK3PXP
Google:user@gmail.com,NRXW4ZDFNZYQ====
```

- **account**：格式为 `服务商:账户名`（如果没有服务商则只显示账户名）
- **secret**：Base32 编码的 OTP 密钥

## 安全提示

- 输出文件权限为 `0600`（仅文件所有者可读写）
- 不使用 `-o` 时会打印警告，建议始终输出到文件
- 妥善保管导出的 CSV 文件，泄露密钥等于泄露二次验证

## License

MIT
