import copy
import importlib.util
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location('ci_resume', ROOT / 'scripts/ci-resume.py')
RESUME = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(RESUME)


def job(name, number, attempt, conclusion, status='completed'):
    return dict(name=name, id=number, run_attempt=attempt, conclusion=conclusion, status=status)


def repaired_jobs():
    return [job('core', 1, 1, 'failure'), job('language', 2, 1, 'cancelled'),
            job('runtime', 3, 1, 'success'), job('test', 4, 1, 'failure'),
            job('core', 5, 2, 'success'), job('test', 6, 2, 'failure')]


class Request:
    def __init__(self, jobs=None):
        self.jobs = repaired_jobs() if jobs is None else jobs
        self.run = dict(id=41, event='pull_request', run_attempt=2, head_sha='abc',
                        status='completed', pull_requests=[{'number': 13}])
        self.pull = dict(state='open', head={'sha': 'abc'})
        self.calls = []
        self.changed_on_recheck = False

    def __call__(self, method, path):
        self.calls.append((method, path))
        if method == 'POST':
            return {}
        if '/jobs?' in path:
            page = int(path.rsplit('=', 1)[1])
            return {'jobs': self.jobs[(page - 1) * 100:page * 100]}
        if '/pulls/13' in path:
            return self.pull
        if '/commits/' in path:
            return [{'number': 13}]
        run = copy.deepcopy(self.run)
        if self.changed_on_recheck and sum('/jobs?' not in p and '/runs/' in p for m, p in self.calls) > 1:
            run['run_attempt'] += 1
        return run

    def posts(self):
        return [path for method, path in self.calls if method == 'POST']


class RetryTests(unittest.TestCase):
    def test_repaired_failure_resumes_cancelled_siblings_despite_incomplete_aggregate(self):
        self.assertEqual(RESUME.resume_plan(repaired_jobs(), 2), ['language'])

    def test_failed_or_cancelled_retry_does_not_resume(self):
        for conclusion in ('failure', 'timed_out', 'cancelled'):
            with self.subTest(conclusion=conclusion):
                jobs = repaired_jobs()
                jobs[4]['conclusion'] = conclusion
                self.assertEqual(RESUME.resume_plan(jobs, 2), [])

    def test_another_unresolved_failure_prevents_resume(self):
        jobs = repaired_jobs()
        jobs[2]['conclusion'] = 'failure'
        self.assertEqual(RESUME.resume_plan(jobs, 2), [])

    def test_successful_job_rerun_does_not_retry_unfinished_work(self):
        jobs = repaired_jobs()
        jobs[0]['conclusion'] = 'success'
        self.assertEqual(RESUME.resume_plan(jobs, 2), [])

    def test_completed_resume_does_not_loop(self):
        jobs = repaired_jobs() + [job('language', 7, 3, 'success')]
        self.assertEqual(RESUME.resume_plan(jobs, 3), [])

    def test_current_running_or_cancelled_job_prevents_resume(self):
        for record in (job('other', 7, 2, None, 'in_progress'), job('other', 7, 2, 'cancelled')):
            with self.subTest(record=record):
                self.assertEqual(RESUME.resume_plan(repaired_jobs() + [record], 2), [])

    def test_resume_posts_only_for_current_completed_open_pr_attempt(self):
        request = Request()
        result = RESUME.resume(request, 'owner/repo', {'workflow_run': request.run.copy()})
        self.assertTrue(result['resumed'])
        self.assertEqual(request.posts(), ['repos/owner/repo/actions/runs/41/rerun-failed-jobs'])
        self.assertEqual(result['jobs'], ['language'])

    def test_stale_closed_or_manually_cancelled_attempt_does_not_post(self):
        for case in ('push', 'first_attempt', 'running', 'new_attempt', 'new_sha', 'closed_pr', 'new_pr_sha', 'recheck_race'):
            with self.subTest(case=case):
                request = Request()
                event = request.run.copy()
                if case == 'push': event['event'] = 'push'
                if case == 'first_attempt': event['run_attempt'] = 1
                if case == 'running': request.run['status'] = 'in_progress'
                if case == 'new_attempt': request.run['run_attempt'] = 3
                if case == 'new_sha': request.run['head_sha'] = 'def'
                if case == 'closed_pr': request.pull['state'] = 'closed'
                if case == 'new_pr_sha': request.pull['head']['sha'] = 'def'
                if case == 'recheck_race': request.changed_on_recheck = True
                self.assertFalse(RESUME.resume(request, 'owner/repo', {'workflow_run': event})['resumed'])
                self.assertEqual(request.posts(), [])

    def test_missing_pull_association_uses_commit_lookup(self):
        request = Request()
        request.run['pull_requests'] = []
        self.assertTrue(RESUME.resume(request, 'owner/repo', {'workflow_run': request.run.copy()})['resumed'])

    def test_aggregate_remains_failed_until_cancelled_jobs_pass(self):
        request = Request()
        self.assertFalse(RESUME.check(request, 'owner/repo', 41)['complete'])
        request.jobs += [job('language', 7, 3, 'success')]
        result = RESUME.check(request, 'owner/repo', 41)
        self.assertTrue(result['complete'])
        self.assertEqual(result['checked'], 3)

    def test_empty_run_never_passes(self):
        self.assertFalse(RESUME.check(Request([]), 'owner/repo', 41)['complete'])

    def test_optional_failures_do_not_block_required_gate(self):
        jobs = [job('core', 1, 1, 'success'), job('Native pause model (Swift, optional)', 2, 1, 'failure')]
        self.assertTrue(RESUME.check(Request(jobs), 'owner/repo', 41)['complete'])

    def test_job_history_is_paginated(self):
        request = Request([job(str(i), i, 1, 'success') for i in range(101)])
        self.assertEqual(len(RESUME.run_jobs(request, 'owner/repo', 41)), 101)
        self.assertEqual(len(request.calls), 2)


if __name__ == '__main__':
    unittest.main()
