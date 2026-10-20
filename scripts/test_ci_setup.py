import contextlib
import importlib.util
import io
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('ci_suite', ROOT / 'scripts/ci-suite.py')
ci = importlib.util.module_from_spec(spec)
spec.loader.exec_module(ci)


class SetupTests(unittest.TestCase):
    def test_fixture_build_cache_is_shared_only_within_one_suite(self):
        for fail in (False, True):
            with self.subTest(fail=fail):
                caches = []

                def run(command, *, cwd, env, check):
                    cache = Path(env['GOALCHEMY_TEST_BUILD_CACHE'])
                    self.assertTrue(cache.is_dir())
                    self.assertEqual(cwd, ROOT)
                    self.assertTrue(check)
                    caches.append(cache)
                    if fail:
                        raise subprocess.CalledProcessError(1, command)

                with mock.patch('sys.argv', ['ci-suite.py', 'pause-swift']), \
                        mock.patch.object(ci.subprocess, 'run', side_effect=run), \
                        contextlib.redirect_stdout(io.StringIO()):
                    if fail:
                        with self.assertRaises(subprocess.CalledProcessError):
                            ci.main()
                    else:
                        ci.main()
                self.assertEqual(len(caches), 1 if fail else 4)
                self.assertEqual(len(set(caches)), 1)
                self.assertFalse(caches[0].exists())

    def test_language_dependencies(self):
        for target in ci.TARGET_NAMES:
            with self.subTest(target=target):
                self.assertEqual(set(ci.setup('language', target)['features']), {'go', target})
        self.assertEqual(ci.setup('language', 'python')['apt'], '')
        self.assertEqual(ci.setup('language', 'typescript')['apt'], '')

    def test_pause_dependencies(self):
        self.assertEqual(set(ci.setup('pause')['features']), {'go', 'typescript', 'csharp', 'go-probe'})
        self.assertEqual(set(ci.setup('pause', 'python')['features']), {'go', 'python'})
        for target in ('java', 'rust', 'swift'):
            self.assertEqual(set(ci.setup('pause-' + target)['features']), {'go', target})

    def test_mixed_suites_keep_full_setup(self):
        full = ci.setup()
        for suite in ('core', 'contracts', 'integration'):
            self.assertEqual(ci.setup(suite), full)
        self.assertTrue(set(ci.TARGET_NAMES).issubset(full['features']))
        self.assertTrue({'browser', 'llvm', 'rust-harness', 'go-probe'}.issubset(full['features']))

    def test_fixture_suites_do_not_build_unused_harnesses(self):
        for suite in ('examples', 'floats', 'naming'):
            self.assertEqual(set(ci.setup(suite)['features']), set(ci.TARGET_NAMES))
        self.assertEqual(set(ci.setup('runtime')['features']), set(ci.TARGET_NAMES) | {'rust-harness'})

    def test_memory_keeps_sanitizer_and_both_targets(self):
        selected = ci.setup('memory')
        self.assertEqual(set(selected['features']), {'go', 'c', 'rust', 'llvm'})
        self.assertIn('libgc-dev', selected['apt'].split())
        self.assertIn('cmake', selected['apt'].split())

    def test_invalid_profiles_fail(self):
        for args in (('missing',), ('language', None), ('language', 'tinygo'), ('pause', 'swift')):
            with self.subTest(args=args), self.assertRaises(ValueError):
                ci.setup(*args)

    def test_workflow_outputs(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'outputs'
            subprocess.run(['python3', str(ROOT / 'scripts/ci-suite.py'), 'language', '--target', 'python',
                            '--setup', '--github-output', str(path)], check=True, capture_output=True)
            values = dict(line.split('=', 1) for line in path.read_text().splitlines())
            self.assertEqual(values['python'], 'true')
            for unused in ('swift', 'llvm', 'rust', 'csharp', 'browser', 'rust_harness'):
                self.assertEqual(values[unused], 'false')
            self.assertEqual(values['apt'], '')

    def test_close_events_never_allocate_runners_or_run_pr_code(self):
        for name in ('ci.yml', 'docs.yml', 'api-docs-template.yml'):
            with self.subTest(workflow=name):
                workflow = (ROOT / '.github/workflows' / name).read_text()
                self.assertIn('pull_request_target:\n    types: [closed]', workflow)
                self.assertIn('github.event.pull_request.number || github.ref', workflow)
                self.assertRegex(workflow, r'(?m)^  cancel-in-progress: true$')
                for job in re.split(r'(?m)^  [\w-]+:\n', workflow.split('\njobs:\n', 1)[1])[1:]:
                    condition = re.search(r'(?m)^    if: (.+)$', job)
                    self.assertIsNotNone(condition)
                    self.assertTrue("github.event_name != 'pull_request_target'" in condition[1] or
                                    "github.event_name == 'push'" in condition[1])
                    self.assertIn('      queue: max', job)
                    self.assertIn('      cancel-in-progress: false', job)

    def test_manual_docs_publishing_is_preserved(self):
        workflow = (ROOT / '.github/workflows/docs.yml').read_text()
        self.assertIn('  push:\n    branches: [main]', workflow)
        deploy = workflow.split('\n  deploy:\n', 1)[1]
        self.assertIn("github.event_name == 'workflow_dispatch'", deploy)
        self.assertIn("github.ref == 'refs/heads/main'", deploy)
        for name in ('ci.yml', 'api-docs-template.yml'):
            triggers = (ROOT / '.github/workflows' / name).read_text().split('\npermissions:', 1)[0]
            self.assertIn('  pull_request:', triggers)
            self.assertNotIn('  push:', triggers)

    def test_bootstrap_does_not_prepare_unselected_targets(self):
        for target in ('go', 'typescript'):
            with self.subTest(target=target), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                for file in ('scripts/ci-bootstrap.sh', 'scripts/ci-suite.py', 'toolchains.lock',
                             'internal/frontend/frontend.go'):
                    destination = root / file
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy(ROOT / file, destination)
                tools = root / 'tools'
                tools.mkdir()
                for tool in ('go', 'npm'):
                    path = tools / tool
                    path.write_text('#!/bin/sh\nprintf "%s\\n" "$0 $*" >> "$CI_TEST_LOG"\n')
                    path.chmod(0o755)
                log = root / 'commands'
                env = dict(os.environ, PATH=str(tools) + ':' + os.environ['PATH'], CI_TEST_LOG=str(log),
                           GITHUB_PATH=str(root / 'github-path'), GITHUB_ENV=str(root / 'github-env'))
                subprocess.run(['bash', str(root / 'scripts/ci-bootstrap.sh'), '--suite', 'language',
                                '--target', target, '--with-browser-deps'], env=env, check=True, capture_output=True)
                commands = log.read_text().splitlines()
                self.assertEqual(len(commands), 1 if target == 'go' else 2)
                self.assertIn('go version', commands[0])
                self.assertNotIn('go1.25.14', '\n'.join(commands))
                self.assertFalse((root / '.toolchains').exists())
                if target == 'typescript':
                    self.assertIn('npm ci --ignore-scripts', commands[1])
                    self.assertIn('targets/typescript/node_modules/.bin', (root / 'github-path').read_text())
                self.assertEqual((root / 'github-env').read_text(), '')

    def test_cached_toolchain_needs_matching_pin_and_executable(self):
        lock = (ROOT / 'toolchains.lock').read_text()
        pins = dict(re.findall(r"^(\w+)='([^']*)'", lock, re.M))
        for target, destination, executable, pin in (
            ('dotnet', 'dotnet', 'dotnet', 'DOTNET_SHA256'),
            ('swift', 'swift-' + pins['SWIFT_VERSION'], 'usr/bin/swiftc', 'SWIFT_SHA256'),
            ('llvm', 'llvm-' + pins['LLVM_VERSION'], 'bin/clang', 'LLVM_SHA256'),
        ):
            with self.subTest(target=target), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                (root / 'scripts').mkdir()
                shutil.copy(ROOT / 'scripts/fetch-toolchains.sh', root / 'scripts')
                shutil.copy(ROOT / 'toolchains.lock', root)
                installed = root / '.toolchains' / destination
                binary = installed / executable
                binary.parent.mkdir(parents=True)
                binary.write_text('#!/bin/sh\nexit 0\n')
                binary.chmod(0o755)
                marker = installed / '.goalchemy-sha256'
                marker.write_text(pins[pin] + '\n')
                if target == 'llvm':
                    lib = root / '.toolchains/libtinfo5/lib/x86_64-linux-gnu/libtinfo.so.5'
                    lib.parent.mkdir(parents=True)
                    lib.touch()
                    (root / '.toolchains/libtinfo5/.goalchemy-sha256').write_text(pins['LIBTINFO5_SHA256'])
                tools = root / 'tools'
                tools.mkdir()
                curl = tools / 'curl'
                curl.write_text('#!/bin/sh\nexit 77\n')
                curl.chmod(0o755)
                env = dict(os.environ, PATH=str(tools) + ':' + os.environ['PATH'])
                command = ['bash', str(root / 'scripts/fetch-toolchains.sh'), target]
                subprocess.run(command, env=env, check=True, capture_output=True)
                marker.write_text('stale pin\n')
                self.assertEqual(subprocess.run(command, env=env, capture_output=True).returncode, 77)
                marker.write_text(pins[pin])
                binary.unlink()
                self.assertEqual(subprocess.run(command, env=env, capture_output=True).returncode, 77)


if __name__ == '__main__':
    unittest.main()
