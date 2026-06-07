; Inno Setup script — Telegram Drive (Ổ Đĩa Cloud Ảo)
; Build: iscc.exe td-agent.iss
; Yêu cầu copy san vao thu muc installer/:
;   - td-agent.exe          (build voi: go build -tags "fuse tray")
;   - pwa\                  (noi dung web-pwa\dist)
;   - winfsp-2.0.23075.msi  (tai tu https://winfsp.dev)

#define AppName "O Dia Cloud Ao"
#define AppVersion "0.1.0"
#define AppPublisher "Pham Thanh"
#define WinFspMsi "winfsp-2.0.23075.msi"

[Setup]
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
DefaultDirName={autopf}\TelegramDrive
DefaultGroupName={#AppName}
PrivilegesRequired=admin
OutputBaseFilename=TelegramDriveSetup
Compression=lzma
SolidCompression=yes
WizardStyle=modern
ArchitecturesInstallIn64BitMode=x64compatible

[Files]
Source: "td-agent.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "pwa\*"; DestDir: "{app}\pwa"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#WinFspMsi}"; DestDir: "{tmp}"; Flags: deleteafterinstall

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\td-agent.exe"; Parameters: "--tray"
Name: "{group}\Mo thu muc du lieu"; Filename: "{userappdata}\TelegramVirtualDrive\agent"
Name: "{userdesktop}\{#AppName}"; Filename: "{app}\td-agent.exe"; Parameters: "--tray"; Tasks: desktopicon

[Tasks]
Name: desktopicon; Description: "Tao shortcut tren Desktop"; GroupDescription: "Tuy chon:"

[Run]
Filename: "msiexec.exe"; Parameters: "/i ""{tmp}\{#WinFspMsi}"" /qb ADDLOCAL=ALL"; StatusMsg: "Dang cai WinFsp (driver mount o ao)..."; Flags: waituntilterminated
Filename: "{app}\td-agent.exe"; Parameters: "--tray"; Description: "Khoi dong {#AppName}"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "taskkill.exe"; Parameters: "/IM td-agent.exe /F"; Flags: runhidden

[Code]
// Cảnh báo người dùng đặt TD_AGENT_SESSION_KEY truoc khi chay production.
procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
  begin
    MsgBox('Da cai dat xong.' + #13#10 +
           'Quan trong: dat bien moi truong TD_AGENT_SESSION_KEY (32-byte hex) ' +
           'truoc khi chay production de ma hoa session Telegram.' + #13#10 +
           'Sinh key: openssl rand -hex 32',
           mbInformation, MB_OK);
  end;
end;