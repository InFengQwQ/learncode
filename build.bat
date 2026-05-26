@echo off
echo Building server image...
docker build -t learncode-server -f server\Dockerfile server
if %ERRORLEVEL% neq 0 exit /b %ERRORLEVEL%

echo Building web image...
docker build -t learncode-web -f web\Dockerfile web
if %ERRORLEVEL% neq 0 exit /b %ERRORLEVEL%

echo Starting services...
docker compose up -d
