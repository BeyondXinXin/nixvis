@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

echo 开始构建 NixVis...
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64

echo 清理旧文件...
if exist nixvis.exe del /f nixvis.exe

REM 获取版本信息
for /f "tokens=1-4 delims=/ " %%a in ('date /t') do (set BUILD_DATE=%%a-%%b-%%c)
for /f "tokens=1-2 delims=: " %%a in ('time /t') do (set BUILD_TIME_ONLY=%%a:%%b)
set BUILD_TIME=%BUILD_DATE% %BUILD_TIME_ONLY%

for /f %%i in ('git rev-parse --short=7 HEAD 2^>nul') do set GIT_COMMIT=%%i
if "%GIT_COMMIT%"=="" set GIT_COMMIT=unknown

echo 版本信息:
echo  - 构建时间: %BUILD_TIME%
echo  - Git提交: %GIT_COMMIT%

echo 编译主程序...
go build -ldflags="-s -w -X 'github.com/beyondxinxin/nixvis/internal/util.BuildTime=%BUILD_TIME%' -X 'github.com/beyondxinxin/nixvis/internal/util.GitCommit=%GIT_COMMIT%'" -o nixvis.exe ./cmd/nixvis/main.go

if %errorlevel% equ 0 (
    echo 构建成功! 可执行文件: nixvis.exe
    
    REM 显示文件大小
    for %%A in (nixvis.exe) do (
        set /a SIZE_KB=%%~zA/1024
        echo 文件大小: !SIZE_KB! KB
    )
    
    echo 构建完成，可执行文件已准备就绪
) else (
    echo 构建失败!
    exit /b 1
)