#ifndef MyAppVersion
#define MyAppVersion "dev"
#endif

#define MyAppName "PC Multitool"
#define MyAppExeName "PC-Gear-Calculator-Windows-x64.exe"
#define MyAppPublisher "Xad0"
#define MyAppURL "https://github.com/xadomas2012/Plane-crazy-multitool"

[Setup]
AppId={{8C7A8F5C-7D4D-4F30-9BB8-8D7C0C5B6A11}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}

; Portable-style installation.
; Default location is Downloads\PC-Multitool, but the user can change it.
DefaultDirName={%USERPROFILE}\Downloads\PC-Multitool
DisableDirPage=no
DisableProgramGroupPage=yes

; Do not require administrator privileges.
PrivilegesRequired=lowest

; Keep the installed folder containing only the application EXE.
Uninstallable=no

OutputDir=dist
OutputBaseFilename=PC-Multitool-Setup-v{#MyAppVersion}

Compression=lzma
SolidCompression=yes

WizardStyle=modern

[Files]
Source: "dist\PC-Gear-Calculator-Windows-x64.exe"; \
    DestDir: "{app}"; \
    DestName: "PC-Gear-Calculator.exe"; \
    Flags: ignoreversion

[Run]
Filename: "{app}\PC-Gear-Calculator.exe"; \
    Description: "Launch PC Multitool"; \
    Flags: nowait postinstall skipifsilent
