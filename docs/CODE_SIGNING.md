# Ký số installer Windows (Code Signing)

Publisher: **Innonet Agency - Automation AI Company**

Installer `TelegramDriveSetup.exe` và `td-agent.exe` được CI tự ký nếu có chứng chỉ. Khi chưa có cert, build vẫn chạy nhưng Windows SmartScreen sẽ báo "Unknown publisher".

## 1. Mua chứng chỉ

- Mua **Code Signing Certificate** (ưu tiên **OV/EV**) từ CA: DigiCert, Sectigo, GlobalSign...
- Đăng ký đúng tên pháp nhân: `Innonet Agency - Automation AI Company`.
- EV cert giúp gỡ cảnh báo SmartScreen nhanh hơn (reputation tức thì).

## 2. Xuất file PFX

CA cấp `.pfx` (chứa private key + cert) kèm mật khẩu.

## 3. Nạp vào GitHub Secrets

Trong repo: Settings → Secrets and variables → Actions → New repository secret.

- `CODE_SIGN_PFX_BASE64`: nội dung PFX dạng base64.
- `CODE_SIGN_PFX_PASSWORD`: mật khẩu PFX.

Tạo base64 (PowerShell):

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("cert.pfx")) | Set-Content cert.pfx.b64
```

Dán nội dung `cert.pfx.b64` vào secret `CODE_SIGN_PFX_BASE64`.

## 4. Build

Push tag `v1.0.0` (hoặc push main) → CI tự ký `td-agent.exe` và `TelegramDriveSetup.exe` bằng signtool + timestamp DigiCert.

Nếu chưa nạp secret, CI bỏ qua bước ký (không fail), nhưng file sẽ chưa được tin cậy.

## Lưu ý

- EV cert thường yêu cầu HSM/token; cân nhắc dùng cloud signing (Azure Trusted Signing / DigiCert KeyLocker) nếu không ký được trên CI bằng PFX.
- Đừng commit PFX vào repo. Chỉ dùng GitHub Secrets.
