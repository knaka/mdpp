@REM Executes the shell script with the same name but with a `.sh` extension instead of `.cmd` extension in the same directory or in the `.\tasks\` directory using BusyBox which is installed if it does not exist.

@echo off
setlocal enabledelayedexpansion
@REM BusyBox for Windows https://frippery.org/busybox/index.html
@REM Release Notes https://frippery.org/busybox/release-notes/index.html
@REM Index of /files/busybox https://frippery.org/files/busybox/?C=M;O=D
set ver=FRP-5579-g5749feb35
if "%PROCESSOR_ARCHITECTURE%" == "x86" (
  echo WARNING: Your environment is 32-bit. Not all features are supported. >&2
  set arch=32
) else if "%PROCESSOR_ARCHITECTURE%" == "AMD64" (
  set arch=64u
) else if "%PROCESSOR_ARCHITECTURE%" == "ARM64" (
  set arch=64a
) else (
  exit /b 1 
)
set cmd_name=busybox-w!arch!-!ver!.exe
set cache_dir_path=%USERPROFILE%\.cache\task-sh
if not exist !cache_dir_path! (
  mkdir "!cache_dir_path!"
)
set cmd_path=!cache_dir_path!\!cmd_name!
if not exist !cmd_path! (
  echo Downloading BusyBox for Windows. >&2
  curl.exe --fail --location --output "!cmd_path!" https://frippery.org/files/busybox/!cmd_name! || exit /b !ERRORLEVEL!
)
set "INITIAL_PWD=%CD%"
set "ARG0=%~f0"
set "ARG0BASE=%~n0"
set "PROJECT_DIR=%~dp0"
if not exist "!PROJECT_DIR!" (
  set "PROJECT_DIR=%CD%"
)
set TASKS_DIR=
if exist "!PROJECT_DIR!\!ARG0BASE!.sh" (
  set TASKS_DIR=!PROJECT_DIR!
) else if exist "!PROJECT_DIR!tasks\!ARG0BASE!.sh" (
  set TASKS_DIR=!PROJECT_DIR!tasks
) else (
  echo Cannot find script file for !ARG0! >&2
  exit /b 1
)
set "script_file_path=!TASKS_DIR!\!ARG0BASE!.sh"
set BB_GLOBBING=0
@REM Virtual shell path of BusyBox Ash
set SH=/bin/sh
!cmd_path! sh "!script_file_path!" %* || exit /b !ERRORLEVEL!
endlocal
