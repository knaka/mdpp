@REM BusyBox for Windows https://frippery.org/busybox/index.html
@REM Release Notes https://frippery.org/busybox/release-notes/index.html
@REM Index of /files/busybox https://frippery.org/files/busybox/?C=M;O=D
@REM The version is copied from ./mise.toml by the "versions:sync" task.
@set bb_ver=FRP-5857-g3681e397f

@echo off
setlocal enabledelayedexpansion

if "%PROCESSOR_ARCHITECTURE%" == "x86" (
  echo WARNING: Your environment is 32-bit. Not all features are supported. >&2
  set bb_arch=32
) else if "%PROCESSOR_ARCHITECTURE%" == "AMD64" (
  set bb_arch=64u
) else if "%PROCESSOR_ARCHITECTURE%" == "ARM64" (
  set bb_arch=64a
) else (
  echo ERROR: Unexpected architecture "%PROCESSOR_ARCHITECTURE%". >&2
  exit /b 1 
)

set cmd_name=busybox.exe
set cache_dir_path=%USERPROFILE%\.cache\task-sh\busybox@%bb_ver%
if not exist !cache_dir_path! (
  mkdir "!cache_dir_path!"
)
set cmd_path=!cache_dir_path!\!cmd_name!
if not exist !cmd_path! (
  echo Downloading BusyBox for Windows. >&2
  curl.exe --fail --location --output "!cmd_path!" https://frippery.org/files/busybox/busybox-w!bb_arch!-!bb_ver!.exe || exit /b !ERRORLEVEL!
)
if not exist !cache_dir_path!\sh.exe (
  !cmd_path! --install !cache_dir_path!
)

@REM Shell-command ready (a5f342b)

set "ARG0=%~f0"
set "ARG0BASE=%~n0"
set script_dir_path=%~dp0
set script_file_path=
if exist "!script_dir_path!\!ARG0BASE!.sh" (
  set script_file_path=!script_dir_path!\!ARG0BASE!.sh
) else if exist "!script_dir_path!\!ARG0BASE!" (
  set script_file_path=!script_dir_path!\!ARG0BASE!
) else (
  echo ERROR: Script file not found: !ARG0BASE!.sh >&2
  exit /b 1
)

endlocal ^
& set "ARG0=%ARG0%" & set "ARG0BASE=%ARG0BASE%" ^
& set "BB_GLOBBING=0" & "%cmd_path%" sh "%script_file_path%" %* || exit /b %ERRORLEVEL%
