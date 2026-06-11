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

[Tasks]
Name: desktopicon; Description: "Tao shortcut tren Desktop"; GroupDescription: "Tuy chon:"
Name: autostart; Description: "Tu khoi dong cung Windows (mount o ao khi dang nhap)"; GroupDescription: "Tuy chon:"

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\td-agent.exe"; Parameters: "--tray --mount-on-start"
Name: "{group}\Mo thu muc du lieu"; Filename: "{userappdata}\TelegramVirtualDrive\agent"
Name: "{userdesktop}\{#AppName}"; Filename: "{app}\td-agent.exe"; Parameters: "--tray --mount-on-start"; Tasks: desktopicon

[Registry]
; Autostart cung Windows: chay tray + tu mount o ao khi user dang nhap.
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "TelegramDrive"; ValueData: """{app}\td-agent.exe"" --tray --mount-on-start"; Tasks: autostart; Flags: uninsdeletevalue

[Run]
Filename: "msiexec.exe"; Parameters: "/i ""{tmp}\{#WinFspMsi}"" /qb ADDLOCAL=ALL"; StatusMsg: "Dang cai WinFsp (driver mount o ao)..."; Flags: waituntilterminated
Filename: "{app}\td-agent.exe"; Parameters: "--tray --mount-on-start"; Description: "Khoi dong {#AppName}"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "taskkill.exe"; Parameters: "/IM td-agent.exe /F"; Flags: runhidden

[Code]
const
  SessionKeyEnv = 'TD_AGENT_SESSION_KEY';

// generateHexKey returns a 64-char hex string (32 random bytes).
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

// Persist a session key to a user-level environment variable the first time,
// so the encrypted Telegram session keeps working across reboots without the
// user having to manage the key manually.
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
    MsgBox('Da cai dat xong.' + #13#10 +
           'Khoa ma hoa session Telegram (TD_AGENT_SESSION_KEY) da duoc tao tu dong ' +
           'va luu trong bien moi truong nguoi dung. Khong xoa bien nay neu khong se phai dang nhap Telegram lai.' + #13#10 +
           'O ao se tu mount khi ban dang nhap Windows (neu da chon Tu khoi dong).',
           mbInformation, MB_OK);
  end;
end;