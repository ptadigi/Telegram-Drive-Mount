# Ký số self-signed + Xác minh checksum

Publisher: **Innonet Agency - Automation AI Company**

> Lưu ý quan trọng: chứng chỉ **self-signed KHÔNG gỡ được cảnh báo Windows
> SmartScreen "Unknown publisher"** trên máy người khác. Nó chỉ chứng minh file
> không bị sửa sau khi ký, và dùng tốt cho nội bộ/đội ngũ. Muốn gỡ hẳn cảnh báo
> phải dùng cert OV/EV của CA (mua) hoặc SignPath Foundation (free cho OSS, cần
> duyệt). Xem cuối tài liệu.

## A. Tạo chứng chỉ self-signed (free)

Trên Windows PowerShell (quyền admin):

```powershell
$cert = New-SelfSignedCertificate `
  -Type CodeSigningCert `
  -Subject "CN=Innonet Agency - Automation AI Company, O=Innonet Agency, C=VN" `
  -KeyAlgorithm RSA -KeyLength 3072 `
  -CertStoreLocation Cert:\CurrentUser\My `
  -NotAfter (Get-Date).AddYears(5)

$pwd = ConvertTo-SecureString -String "DAT-MAT-KHAU-MANH" -Force -AsPlainText
Export-PfxCertificate -Cert $cert -FilePath "$HOME\td-codesign.pfx" -Password $pwd
```

## B. Nạp vào GitHub Secrets

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("$HOME\td-codesign.pfx")) | Set-Content "$HOME\td-codesign.b64"
```

Trong repo: Settings → Secrets and variables → Actions:
- `CODE_SIGN_PFX_BASE64`: dán nội dung file `.b64`.
- `CODE_SIGN_PFX_PASSWORD`: mật khẩu PFX.

Push tag `v*` → CI tự ký `td-agent.exe` và `TelegramDriveSetup.exe` (signtool +
timestamp). Nếu chưa nạp secret, CI bỏ qua bước ký, không fail.

## C. Xác minh bản tải (cho người dùng cộng đồng)

Mỗi release có file `SHA256SUMS.txt`. Người dùng tự kiểm tra:

```powershell
Get-FileHash .\TelegramDriveSetup.exe -Algorithm SHA256
# So sánh với dòng tương ứng trong SHA256SUMS.txt
```

Trùng hash = file nguyên vẹn, không bị chèn mã độc.

## D. Vì sao vẫn còn cảnh báo SmartScreen

- Self-signed: máy người khác không có cert trong Trusted Root → vẫn "Unknown publisher".
- Cách user mở: bấm **More info → Run anyway**.
- Reputation SmartScreen tăng dần theo lượt tải, nhưng chậm và không chắc.

## E. Khi muốn gỡ hẳn cảnh báo (sau này)

1. **SignPath Foundation** — free cho OSS, ký qua service (không dùng PFX secret).
   Cần đăng ký + duyệt tại signpath.org/teams. Khi được duyệt, workflow cần
   chỉnh sang dùng SignPath action thay cho signtool + PFX.
2. **Certum Open Source** — ~30 USD/năm, cert thật, gỡ được cảnh báo.
3. **DigiCert/Sectigo OV/EV** — đắt hơn, EV gỡ cảnh báo nhanh nhất.
