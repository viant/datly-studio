"""Render a fixture file for browser upload; this does not mutate Studio."""
from pathlib import Path
import re
import sys
source, destination, component = sys.argv[1:]
if not re.fullmatch(r'[A-Za-z0-9_-]+', component):
    raise SystemExit('Invalid component identity for DQL fixture')
Path(destination).write_text(Path(source).read_text().replace('__REPORT_ID__', component))
