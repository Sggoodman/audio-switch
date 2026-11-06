; Audio Switch Windows 安装脚本（NSIS MUI2，简体中文）
; 由 scripts/build.sh windows-app 调用:
;   makensis -DVERSION=x.y.z -DVERNAME=vx.y.z scripts/installer.nsi
; 路径使用正斜杠，macOS(makensis) 与 Windows 均可编译

!include "MUI2.nsh"

!ifndef VERSION
  !define VERSION "0.0.0"
!endif
!ifndef VERNAME
  !define VERNAME "v${VERSION}"
!endif

Name "Audio Switch"
OutFile "build/Audio-Switch-${VERNAME}-setup.exe"
Unicode true
InstallDir "$PROGRAMFILES64\Audio Switch"
InstallDirRegKey HKCU "Software\Audio Switch" "InstallDir"
RequestExecutionLevel admin

!define MUI_ABORTWARNING
!define MUI_ICON "assets/Icon.ico"
!define MUI_UNICON "assets/Icon.ico"

; 安装页面：欢迎(隐含) → 选择路径 → 安装
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
; 卸载页面
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "SimpChinese"

Section "Install"
    SetOutPath "$INSTDIR"
    File "build/audio-switch.exe"
    File "build/Icon.png"
    File "build/Icon.ico"

    WriteUninstaller "$INSTDIR\Uninstall.exe"

    ; 快捷方式
    CreateDirectory "$SMPROGRAMS\Audio Switch"
    CreateShortcut "$SMPROGRAMS\Audio Switch\Audio Switch.lnk" "$INSTDIR\audio-switch.exe"
    CreateShortcut "$SMPROGRAMS\Audio Switch\卸载 Audio Switch.lnk" "$INSTDIR\Uninstall.exe"
    CreateShortcut "$DESKTOP\Audio Switch.lnk" "$INSTDIR\audio-switch.exe"

    ; 卸载信息（控制面板可见）
    WriteRegStr HKCU "Software\Audio Switch" "InstallDir" "$INSTDIR"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Audio Switch" "DisplayName" "Audio Switch"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Audio Switch" "DisplayVersion" "${VERSION}"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Audio Switch" "DisplayIcon" "$INSTDIR\audio-switch.exe"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Audio Switch" "UninstallString" "$INSTDIR\Uninstall.exe"
    WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Audio Switch" "NoModify" 1
    WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Audio Switch" "NoRepair" 1

    ; 开机自启由应用内「设置 → 开机自动启动」开关管理（写 HKCU Run，路径自动跟随安装目录）
SectionEnd

Section "Uninstall"
    ; 先结束运行中的进程（忽略失败）
    ExecWait "taskkill /IM audio-switch.exe /F"

    Delete "$DESKTOP\Audio Switch.lnk"
    RMDir /r "$SMPROGRAMS\Audio Switch"
    DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Audio Switch"
    DeleteRegKey HKCU "Software\Audio Switch"
    DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "AudioSwitch"

    Delete "$INSTDIR\audio-switch.exe"
    Delete "$INSTDIR\Icon.png"
    Delete "$INSTDIR\Icon.ico"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir "$INSTDIR"
SectionEnd
