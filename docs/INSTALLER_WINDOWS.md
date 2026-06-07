# Cài đặt Telegram-Drive-Mount trên Windows

Bản hướng dẫn nhanh để đóng gói agent + WinFsp thành 1 installer Windows duy nhất.

## Yêu cầu

- Đã build `td-agent.exe` với `-tags fuse`.
- `web-pwa/dist` đã build (`npm run build`).
- WinFsp installer `winfsp-2.0.23075.msi` (hoặc bản mới hơn từ https://winfsp.dev).
- [Inno Setup 6](https://jrsoftware.org/isdl.php) cài trên máy build.

## Bước 1. Cấu trúc thư mục

```
installer/
├── td-agent.exe                  # build từ agent-go với -tags fuse
├── pwa/                          # nội dung web-pwa/dist
├── winfsp-2.0.23075.msi
└── td-agent-installer.iss
```

## Bước 2. File `td-agent-installer.iss`

```ini
[Setup]
AppName=Ổ Đĩa Cloud Ảo
AppVersion=0.1.0
AppPublisher=Phạm Thành
DefaultDirName={autopf}\TelegramDrive
DefaultGroupName=Ổ Đĩa Cloud Ảo
PrivilegesRequired=admin
OutputBaseFilename=TelegramDriveSetup
Compression=lzma
SolidCompression=yes
WizardStyle=modern

[Files]
Source: "td-agent.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "pwa\*"; DestDir: "{app}\pwa"; Flags: ignoreversion recursesubdirs
Source: "winfsp-2.0.23075.msi"; DestDir: "{tmp}"; Flags: deleteafterinstall

[Icons]
Name: "{group}\Ổ Đĩa Cloud Ảo"; Filename: "{app}\td-agent.exe"; Parameters: "--tray"
Name: "{group}\Mở thư mục dữ liệu"; Filename: "{userappdata}\TelegramVirtualDrive\agent"
Name: "{userdesktop}\Ổ Đĩa Cloud Ảo"; Filename: "{app}\td-agent.exe"; Parameters: "--tray"; Tasks: desktopicon

[Tasks]
Name: desktopicon; Description: "Tạo shortcut trên Desktop"; GroupDescription: "Tuỳ chọn:"

[Run]
Filename: "msiexec.exe"; Parameters: "/i ""{tmp}\winfsp-2.0.23075.msi"" /qb ADDLOCAL=ALL"; StatusMsg: "Đang cài WinFsp..."; Flags: waituntilterminated
Filename: "{app}\td-agent.exe"; Parameters: "--tray"; Description: "Khởi động Ổ Đĩa Cloud Ảo"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "taskkill.exe"; Parameters: "/IM td-agent.exe /F"; Flags: runhidden
```

## Bước 3. Build installer

1. Mở Inno Setup → Open → chọn `td-agent-installer.iss`.
2. Build → Compile (Ctrl+F9). Output `Output/TelegramDriveSetup.exe`.
3. Copy file ra ngoài để phát hành.

## Bước 4. Test

- Chạy `TelegramDriveSetup.exe` trên máy chưa cài WinFsp.
- Approve UAC.
- Sau khi xong, mở Start Menu → Ổ Đĩa Cloud Ảo.
- Kiểm tra agent listen `127.0.0.1:8750`, mount drive `T:` từ tray.

## Ghi chú

- Nếu user đã có WinFsp, msiexec sẽ idempotent, không cần handle riêng.
- Khi update version, đổi `AppVersion` và file MSI WinFsp tương ứng.
- Code-signing executable nên dùng EV cert + signtool trước khi đóng gói.
- Logo `.ico` đặt vào `[Setup] SetupIconFile=app.ico` cho đẹp.
- Auto-start cùng máy đã có sẵn trong tray menu (registry `HKCU\Run`).

## Khi muốn build lại từ source

```powershell
# Build agent
cd agent-go
$env:PATH = "C:\msys64\mingw64\bin;$env:PATH"
$env:CGO_ENABLED = "1"
$env:CPATH = "C:\Program Files (x86)\WinFsp\inc\fuse"
go build -tags "fuse tray" -o ..\installer\td-agent.exe .\cmd\agent

# Build PWA
cd ..\web-pwa
npm install
npm run build
xcopy /E /I /Y dist ..\installer\pwa

# Compile installer trong Inno Setup
"C:\Program Files (x86)\Inno Setup 6\ISCC.exe" ..\installer\td-agent.iss
```

> File `installer/td-agent.iss` đã có sẵn trong repo, không cần copy từ docs nữa.
