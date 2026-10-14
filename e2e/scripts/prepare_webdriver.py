"""Install a ChromeDriver matching the local Chrome build into the e2e directory."""
import io
import json
import os
from pathlib import Path
import platform
import re
import shutil
import subprocess
import sys
import urllib.request
import zipfile

root = Path(sys.argv[1]).resolve()
system, arch = platform.system(), platform.machine()
if system == 'Darwin':
    browser = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'
    target = 'mac-arm64' if arch == 'arm64' else 'mac-x64'
else:
    browser = shutil.which('google-chrome') or shutil.which('chromium')
    target = 'linux-arm64' if arch in ('arm64', 'aarch64') else 'linux64'
if not browser:
    raise SystemExit('Chrome is required for Endly WebDriver acceptance tests')
version = re.search(r'\d+\.\d+\.\d+\.\d+', subprocess.check_output([browser, '--version'], text=True)).group()
endpoint = 'https://googlechromelabs.github.io/chrome-for-testing/latest-patch-versions-per-build-with-downloads.json'
with urllib.request.urlopen(endpoint, timeout=30) as response:
    builds = json.load(response)['builds']
release = builds.get('.'.join(version.split('.')[:3]))
if not release:
    raise SystemExit('No official ChromeDriver release for installed Chrome ' + version)
artifact = next((a for a in release['downloads']['chromedriver'] if a['platform'] == target), None)
if not artifact:
    raise SystemExit('No official ChromeDriver artifact for ' + target)
root.mkdir(parents=True, exist_ok=True)
destination = root / 'chromedriver'
stamp = root / 'version.txt'
if not destination.exists() or not stamp.exists() or stamp.read_text() != release['version']:
    with urllib.request.urlopen(artifact['url'], timeout=60) as response:
        archive = zipfile.ZipFile(io.BytesIO(response.read()))
    member = next(n for n in archive.namelist() if n.endswith('/chromedriver'))
    temporary = root / 'chromedriver.download'
    temporary.write_bytes(archive.read(member))
    temporary.chmod(0o755)
    os.replace(temporary, destination)
    stamp.write_text(release['version'])
print('Chrome ' + version + '; ChromeDriver ' + release['version'])
