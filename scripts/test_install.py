"""Offline installer checks; no network requests or real installation changes."""
import hashlib
import io
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest

INSTALLER = Path(__file__).resolve().parents[1] / "install"

class InstallerTests(unittest.TestCase):
    def run_installer(self, mode="ok", version="0.1.0", arch="x86_64", system="Linux"):
        with tempfile.TemporaryDirectory() as root:
            root = Path(root)
            fixture = root / "fixture.tar.gz"
            with tarfile.open(fixture, "w:gz") as archive:
                data = b"#!/bin/sh\necho fixture\n"
                info = tarfile.TarInfo("opencode")
                info.size, info.mode = len(data), 0o755
                archive.addfile(info, io.BytesIO(data))
            digest = hashlib.sha256(fixture.read_bytes()).hexdigest()
            fakebin = root / "bin"
            fakebin.mkdir()
            curl = fakebin / "curl"
            curl.write_text('''#!/usr/bin/env python3
import os,sys,shutil
from pathlib import Path
args=sys.argv[1:]
if os.environ['TEST_MODE']=='download_fail': sys.exit(22)
url=next(a for a in args if a.startswith('https://'))
if '/releases/latest' in url:
 print('{"tag_name":"v0.1.0"}' if os.environ['TEST_MODE']!='bad_metadata' else '{}')
 sys.exit(0)
out=Path(args[args.index('-o')+1])
if url.endswith('checksums.txt'):
 digest='0'*64 if os.environ['TEST_MODE']=='bad_checksum' else os.environ['TEST_DIGEST']
 out.write_text(digest+'  '+os.environ['TEST_FILENAME']+'\\n')
else: shutil.copyfile(os.environ['TEST_ARCHIVE'],out)
''')
            curl.chmod(0o755)
            uname = fakebin / "uname"
            uname.write_text(f'#!/bin/sh\ncase "$1" in -s) echo {system};; -m) echo {arch};; esac\n')
            uname.chmod(0o755)
            target = root / "install path"
            target.mkdir()
            binary = target / "opencode"
            binary.write_text("existing installation")
            temp = root / "tmp"
            temp.mkdir()
            os_name = "mac" if system == "Darwin" else "linux"
            cpu = "arm64" if arch in ("arm64", "aarch64") else "x86_64"
            env = dict(os.environ, PATH=str(fakebin)+os.pathsep+os.environ["PATH"],
                       OPENCODE_INSTALL_DIR=str(target), VERSION=version, TMPDIR=str(temp),
                       TEST_MODE=mode, TEST_DIGEST=digest, TEST_FILENAME=f"opencode-{os_name}-{cpu}.tar.gz",
                       TEST_ARCHIVE=str(fixture))
            result = subprocess.run(["bash",str(INSTALLER)],env=env,capture_output=True,text=True,timeout=10)
            self.assertEqual(list(temp.iterdir()),[],"temporary downloads must be cleaned up")
            return result, binary.read_text()

    def test_versioned_install_and_paths_with_spaces(self):
        result, data = self.run_installer(version="v0.1.0")
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertIn("fixture",data)

    def test_latest_install(self):
        result, _ = self.run_installer(version="")
        self.assertEqual(result.returncode,0,result.stderr)

    def test_mac_arm64_asset(self):
        result, _ = self.run_installer(system="Darwin",arch="arm64")
        self.assertEqual(result.returncode,0,result.stderr)

    def test_failures_preserve_install(self):
        for mode, version in [("download_fail","0.1.0"),("bad_metadata",""),("bad_checksum","0.1.0"),("ok","../../bad")]:
            with self.subTest(mode=mode,version=version):
                result,data=self.run_installer(mode=mode,version=version)
                self.assertNotEqual(result.returncode,0)
                self.assertEqual(data,"existing installation")

if __name__ == "__main__":
    unittest.main()
