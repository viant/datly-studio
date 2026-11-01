#!/usr/bin/env python3
"""Run native release regressions on an owned, disposable MySQL instance."""
import os
from pathlib import Path
import secrets
import subprocess
import time
import uuid


def main():
    root = Path(__file__).resolve().parents[2]
    name = "datly-studio-release-e2e-" + uuid.uuid4().hex[:12]
    password = secrets.token_hex(24)
    env = {**os.environ, "MYSQL_ROOT_PASSWORD": password}
    started = False
    try:
        subprocess.run(
            ["docker", "run", "--detach", "--rm", "--name", name,
             "--label", "datly-studio.e2e.owner=" + name,
             "-e", "MYSQL_ROOT_PASSWORD", "-e", "MYSQL_DATABASE=studio_e2e",
             "-p", "127.0.0.1::3306", "mysql:8.4"],
            env=env, check=True, stdout=subprocess.DEVNULL, timeout=180)
        started = True
        deadline = time.monotonic() + 120
        while True:
            ready = subprocess.run(
                ["docker", "exec", name, "sh", "-c",
                 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --protocol=TCP -h127.0.0.1 -uroot studio_e2e --execute="SELECT 1"'],
                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10)
            if ready.returncode == 0:
                break
            if time.monotonic() >= deadline:
                raise RuntimeError("owned MySQL fixture did not become ready")
            time.sleep(1)
        binding = subprocess.check_output(["docker", "port", name, "3306/tcp"], text=True).strip()
        if not binding.startswith("127.0.0.1:") or "\n" in binding:
            raise RuntimeError("MySQL fixture is not bound exclusively to loopback")
        port = int(binding.rsplit(":", 1)[1])
        test_env = {**os.environ, "GOWORK": "off", "STUDIO_RELEASE_MYSQL_DSN":
                    f"root:{password}@tcp(127.0.0.1:{port})/studio_e2e?parseTime=true&multiStatements=true"}
        subprocess.run(
            ["docker", "exec", "-i", name, "sh", "-c",
             'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot'],
            input=b"CREATE DATABASE studio_snapshot_e2e; CREATE DATABASE studio_audit_e2e; CREATE DATABASE studio_session_e2e; CREATE DATABASE studio_resource_e2e;", check=True,
            stdout=subprocess.DEVNULL, timeout=10)
        test_env["STUDIO_RELEASE_SNAPSHOT_MYSQL_DSN"] = (
            f"root:{password}@tcp(127.0.0.1:{port})/studio_snapshot_e2e?parseTime=true&multiStatements=true")
        test_env["STUDIO_AUDIT_MYSQL_DSN"] = (
            f"root:{password}@tcp(127.0.0.1:{port})/studio_audit_e2e?parseTime=true&multiStatements=true")
        test_env["STUDIO_SESSION_MYSQL_DSN"] = (
            f"root:{password}@tcp(127.0.0.1:{port})/studio_session_e2e?parseTime=true&multiStatements=true")
        test_env["STUDIO_RESOURCE_MYSQL_DSN"] = (
            f"root:{password}@tcp(127.0.0.1:{port})/studio_resource_e2e?parseTime=true&multiStatements=true")
        result = subprocess.run(
            ["go", "test", "-p=1", "./studio/artifact_release/channels",
             "./studio/artifact_release/snapshot", "./internal/auditstore", "./internal/bffauth", "./runtime/resources", "-run", "MySQL", "-count=1", "-v"],
            cwd=root, env=test_env, text=True, stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT, timeout=300)
        # Fixture credentials are never part of diagnostic output.
        print(result.stdout.replace(password, "[redacted]"), end="")
        required = ("TestNativeChannelsMySQL", "TestNativeSnapshotGraphMySQL", "TestNativeAuditControlsMySQL", "TestNativeRefreshLeaseMySQL", "TestResourceSnapshotMySQL")
        if result.returncode or "--- SKIP:" in result.stdout or any(
                "--- PASS: " + name + " " not in result.stdout for name in required):
            raise RuntimeError("native MySQL release qualification did not pass without skips")
    finally:
        if started:
            subprocess.run(["docker", "rm", "--force", name],
                           stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=30, check=True)


if __name__ == "__main__":
    main()
