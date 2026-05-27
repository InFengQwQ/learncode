@echo off
setlocal

docker build -t learncode-server -f server\Dockerfile server || (
	set "EXITCODE=%ERRORLEVEL%" & goto :end
)

docker build -t learncode-web -f web\Dockerfile web || (
	set "EXITCODE=%ERRORLEVEL%" & goto :end
)

docker compose up -d || (
	set "EXITCODE=%ERRORLEVEL%" & goto :end
)

rem all good
set "EXITCODE=0"

:end
timeout /t 5 /nobreak >nul
exit /b %EXITCODE%
