<#
.SYNOPSIS
Create or reuse a local subtitle test session, then start the desktop client.
.DESCRIPTION
Uses the adjacent server's running Docker container and existing database models.
Does not edit server files or change authentication. New test accounts receive
16,500 development credits; existing balances are never reset or replenished.
#>
[CmdletBinding()]
param(
    [string]$ServerDirectory = '',
    [string]$ApiBaseUrl = 'http://127.0.0.1:8000',
    [ValidateRange(1, 48)][int]$Hours = 24,
    [switch]$RefreshToken,
    [switch]$NoLaunch,
    [switch]$ScreenshotDebug
)

$ErrorActionPreference = 'Stop'
$scriptDirectory = Split-Path -Parent $MyInvocation.MyCommand.Path
if (-not $ServerDirectory) { $ServerDirectory = Join-Path $scriptDirectory '..\..\lychee-server' }
$apiUri = [Uri]$ApiBaseUrl
if (-not $apiUri.IsAbsoluteUri -or -not $apiUri.IsLoopback -or $apiUri.Scheme -notin @('http', 'https')) {
    throw 'This helper is only for a local development API (localhost / 127.0.0.1).'
}
$serverPath = (Resolve-Path -LiteralPath $ServerDirectory).Path
$composePath = Join-Path $serverPath 'compose.yaml'
if (-not (Test-Path -LiteralPath $composePath)) { throw "Missing Docker configuration: $composePath" }
$baseUrl = $ApiBaseUrl.TrimEnd('/')
try {
    $null = Invoke-RestMethod -Uri "$baseUrl/health/ready" -TimeoutSec 10
} catch {
    throw 'The local server is not ready. Start it with docker compose up -d in lychee-server.'
}

$testIdentity = 'local-subtitle-test'
$token = [Environment]::GetEnvironmentVariable('LYCHEE_ACCESS_TOKEN', 'User')
$profile = $null
$expiresAt = $null
if ($token -and -not $RefreshToken) {
    try {
        $candidate = Invoke-RestMethod -Uri "$baseUrl/v1/users/me" -Headers @{ Authorization = "Bearer $token" } -TimeoutSec 10
        if ($candidate.steam_id -eq $testIdentity) { $profile = $candidate }
    } catch {
        # Missing / expired sessions are replaced below; never print the token.
    }
}

if ($null -eq $profile) {
    # Send this administrative setup through stdin, without adding a script to
    # the server checkout or exposing the new token on the command line.
    $python = @'
import asyncio
import hashlib
import json
import secrets
import sys
from datetime import UTC, datetime, timedelta

from sqlalchemy import delete, inspect, select
from app.core.config import Settings
from app.db.database import Database
from app.db.models import User, UserCredits, UserSession
from app.services.users import INITIAL_CREDITS

async def main():
    database = Database(Settings())
    now = datetime.now(UTC)
    expires = now + timedelta(hours=int(sys.argv[1]))
    token = secrets.token_urlsafe(32)
    try:
        async with database.engine.connect() as connection:
            tables = await connection.run_sync(lambda c: set(inspect(c).get_table_names()))
        if not {'users', 'user_credits', 'user_sessions'} <= tables:
            raise RuntimeError('Database tables are missing. Run docker compose exec -T api alembic upgrade head first.')
        async with database.sessions.begin() as session:
            user = await session.scalar(
                select(User).where(User.steam_id == 'local-subtitle-test').with_for_update()
            )
            created = user is None
            if created:
                # A recognizable development marker, never a real Steam identity.
                user = User(steam_id='local-subtitle-test')
                session.add(user)
                await session.flush()
                credits = UserCredits(user_id=user.id, balance=INITIAL_CREDITS)
                session.add(credits)
            else:
                credits = await session.get(UserCredits, user.id)
                if credits is None:
                    raise RuntimeError('Test credit account is missing; no existing balance was changed')
            await session.execute(
                delete(UserSession).where(
                    UserSession.user_id == user.id,
                    UserSession.expires_at <= now.replace(tzinfo=None),
                )
            )
            session.add(UserSession(
                token_hash=hashlib.sha256(token.encode()).hexdigest(),
                user_id=user.id,
                expires_at=expires.replace(tzinfo=None),
            ))
            result = {
                'access_token': token,
                'expires_at': expires.isoformat(),
                'created': created,
            }
        # PowerShell captures this JSON in memory; it is never printed to its UI.
        print(json.dumps(result))
    finally:
        await database.close()

try:
    asyncio.run(main())
except Exception as error:
    # Do not dump database connection details or SQL parameters to the terminal.
    message = str(error) if isinstance(error, RuntimeError) else type(error).__name__
    print('Local test session setup failed: ' + message, file=sys.stderr)
    sys.exit(1)
'@
    $raw = $python | & docker compose --project-directory $serverPath -f $composePath exec -T api python - $Hours
    if ($LASTEXITCODE -ne 0) { throw 'Failed to create the local test session. Server files were not changed.' }
    try { $sessionResult = ($raw -join "`n") | ConvertFrom-Json } catch { throw 'The test setup returned an invalid response.' }
    $token = $sessionResult.access_token
    if ($token -notmatch '^[A-Za-z0-9_-]{43}$') { throw 'The test setup did not return a valid session token.' }
    $expiresAt = [DateTimeOffset]::Parse($sessionResult.expires_at).ToLocalTime()
    try {
        $profile = Invoke-RestMethod -Uri "$baseUrl/v1/users/me" -Headers @{ Authorization = "Bearer $token" } -TimeoutSec 10
    } catch { throw 'The newly created session failed API authentication. Check the server logs.' }
    if ($profile.steam_id -ne $testIdentity) { throw 'The API and Docker container are not using the same test account.' }
}

# The desktop client reads its own process environment. Persist for future
# runs, and explicitly set this process before launching the executable.
[Environment]::SetEnvironmentVariable('LYCHEE_ACCESS_TOKEN', $token, 'User')
$env:LYCHEE_ACCESS_TOKEN = $token
if ($ScreenshotDebug) { $env:LYCHEE_SCREENSHOT_DEBUG = '1' }
$env:LYCHEE_SPEECH_URL = ($baseUrl -replace '^http', 'ws') + '/v1/translate/speech'
Write-Host "Test account ready: user_id=$($profile.user_id), credits=$($profile.credits)"
if ($expiresAt) { Write-Host "Session expires: $($expiresAt.ToString('yyyy-MM-dd HH:mm zzz'))" }
Write-Host 'LYCHEE_ACCESS_TOKEN is configured for this process and your Windows user. Its value is hidden.'
if ($profile.credits -le 0) { Write-Warning 'The test account has no credits left. Existing balances have not been changed.' }

if (-not $NoLaunch) {
    $exe = Join-Path $scriptDirectory '..\build\bin\lychee.exe'
    if (-not (Test-Path -LiteralPath $exe)) { throw 'Desktop executable is missing. Run wails build first.' }
    if (Get-Process -Name lychee -ErrorAction SilentlyContinue) {
        Write-Host 'Lychee is already running. Exit it, then run this script again to start with the configured token.'
    } else {
        # This is the interactive desktop app the user is about to test.
        Start-Process -FilePath (Resolve-Path -LiteralPath $exe).Path
    }
}
