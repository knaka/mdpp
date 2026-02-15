@REM Home | mise-en-place https://mise.jdx.dev/
@REM Releases · jdx/mise https://github.com/jdx/mise/releases
@set mise_ver=2026.2.9

@echo off
setlocal enabledelayedexpansion

if "%PROCESSOR_ARCHITECTURE%" == "AMD64" (
  set mise_arch=x64
) else if "%PROCESSOR_ARCHITECTURE%" == "ARM64" (
  set mise_arch=arm64
) else (
  echo ERROR: Unexpected architecture "%PROCESSOR_ARCHITECTURE%". >&2
  exit /b 1 
)

set cache_dir_path=%USERPROFILE%\.cache\task-sh\mise@%mise_ver%
if not exist !cache_dir_path! (
  mkdir "!cache_dir_path!"
)
set cmd_name=mise.exe
set cmd_path=!cache_dir_path!\!cmd_name!
if not exist !cmd_path! (
  echo Downloading Mise for Windows. >&2
  set zip_path=!cache_dir_path!\mise.zip
  curl.exe --fail --location --output "!zip_path!" https://github.com/jdx/mise/releases/download/v!mise_ver!/mise-v!mise_ver!-windows-!mise_arch!.zip || exit /b !ERRORLEVEL!
  unzip.exe "!zip_path!" -d !cache_dir_path!\work || exit /b !ERRORLEVEL!
  move !cache_dir_path!\work\mise\bin\mise.exe "!cache_dir_path!"
  del /q "!zip_path!"
  del /q /s !cache_dir_path!\work
)
!cmd_path! %* || exit /b !ERRORLEVEL!

endlocal ^
& "%cmd_path%" %* || exit /b %ERRORLEVEL%
