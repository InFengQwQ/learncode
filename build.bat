@echo off
echo Building server image...
docker build -t learncode-server -f server\Dockerfile server
if %ERRORLEVEL% neq 0 goto :end

echo Building web image...
docker build -t learncode-web -f web\Dockerfile web
if %ERRORLEVEL% neq 0 goto :end

echo Starting services...
docker compose up -d

:end
set EXITCODE=%ERRORLEVEL%
timeout /t 5 /nobreak
exit /b %EXITCODE%
