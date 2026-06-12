; Inno Setup script — Telegram Drive (Ổ Đĩa Cloud Ảo)
; Build: iscc.exe td-agent.iss
; Yêu cầu copy sẵn vào thư mục installer/:
;   - td-agent.exe          (build với: go build -tags "fuse tray" -ldflags "-H windowsgui")
;   - pwa\                  (nội dung web-pwa\dist)
;   - winfsp-2.0.23075.msi  (tải từ https://winfsp.dev)

#define AppName "Ổ Đĩa Cloud Ảo - Telegram Drive"
#define AppVersion "1.7.7"
#define AppPublisher "Innonet Agency - Automation AI Company"
#define AppURL "https://github.com/ptadigi/Telegram-Drive-Mount"
#define WinFspMsi "winfsp-2.0.23075.msi"

[Setup]
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL={#AppURL}
AppSupportURL={#AppURL}
AppUpdatesURL={#AppURL}
VersionInfoCompany={#AppPublisher}
VersionInfoProductName={#AppName}
VersionInfoVersion={#AppVersion}
VersionInfoCopyright=Copyright (C) Innonet Agency
DefaultDirName={autopf}\TelegramDrive
DefaultGroupName={#AppName}
PrivilegesRequired=admin
OutputBaseFilename=TelegramDriveSetup
Compression=lzma
SolidCompression=yes
WizardStyle=modern
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "vi"; MessagesFile: "compiler:Default.isl"

[Files]
Source: "td-agent.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "pwa\*"; DestDir: "{app}\pwa"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#WinFspMsi}"; DestDir: "{tmp}"; Flags: deleteafterinstall

[Tasks]
Name: desktopicon; Description: "Tạo lối tắt trên Màn hình"; GroupDescription: "Tùy chọn:"
Name: autostart; Description: "Tự khởi động cùng Windows"; GroupDescription: "Tùy chọn:"

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\td-agent.exe"; Parameters: "--tray"
Name: "{group}\Mở thư mục dữ liệu"; Filename: "{userappdata}\TelegramVirtualDrive\agent"
Name: "{userdesktop}\{#AppName}"; Filename: "{app}\td-agent.exe"; Parameters: "--tray"; Tasks: desktopicon

[Registry]
; Tự khởi động cùng Windows: chạy tray, app tự quyết mount theo cấu hình đã lưu.
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "TelegramDrive"; ValueData: """{app}\td-agent.exe"" --tray"; Tasks: autostart; Flags: uninsdeletevalue

[Run]
Filename: "msiexec.exe"; Parameters: "/i ""{tmp}\{#WinFspMsi}"" /qb ADDLOCAL=ALL"; StatusMsg: "Đang cài WinFsp (driver mount ổ ảo)..."; Flags: waituntilterminated
Filename: "{app}\td-agent.exe"; Parameters: "--tray"; Description: "Khởi động {#AppName}"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "taskkill.exe"; Parameters: "/IM td-agent.exe /F"; Flags: runhidden

[Code]
const
  SessionKeyEnv = 'TD_AGENT_SESSION_KEY';

// GenerateHexKey trả về chuỗi hex 64 ký tự (32 byte ngẫu nhiên).
function GenerateHexKey(): string;
var
  i, b, hi, lo: Integer;
  hex, digits: string;
begin
  digits := '0123456789abcdef';
  hex := '';
  for i := 1 to 32 do
  begin
    b := Random(256);
    hi := (b div 16) + 1;
    lo := (b mod 16) + 1;
    hex := hex + Copy(digits, hi, 1) + Copy(digits, lo, 1);
  end;
  Result := hex;
end;

// Lưu khóa session vào biến môi trường người dùng ở lần đầu, để session
// Telegram mã hóa vẫn dùng được qua các lần khởi động mà không cần người
// dùng tự quản lý khóa.
procedure EnsureSessionKey();
var
  existing: string;
  key: string;
begin
  if RegQueryStringValue(HKCU, 'Environment', SessionKeyEnv, existing) and (Length(existing) = 64) then
    exit;
  key := GenerateHexKey();
  RegWriteStringValue(HKCU, 'Environment', SessionKeyEnv, key);
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
  begin
    EnsureSessionKey();
    MsgBox('Đã cài đặt xong.' + #13#10 +
           'Khóa mã hóa session Telegram (TD_AGENT_SESSION_KEY) đã được tạo tự động ' +
           'và lưu trong biến môi trường người dùng. Không xóa biến này, nếu không sẽ phải đăng nhập Telegram lại.' + #13#10 +
           'Mở ứng dụng để cấu hình kết nối tới máy chủ (local hoặc VPS) và ghép thiết bị.',
           mbInformation, MB_OK);
  end;
end;
