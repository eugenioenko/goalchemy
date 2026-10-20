#!/usr/bin/env python3
import argparse
import json
import os
import subprocess


OPTIONAL_JOBS = {'Native pause model (Swift, optional)', 'Native pause model (Rust, optional)'}
FAILED = {'failure', 'timed_out', 'action_required', 'startup_failure'}


def api(method, path):
    result = subprocess.run(['gh', 'api', '--method', method, path],
                            capture_output=True, text=True, check=True, timeout=30)
    return json.loads(result.stdout) if result.stdout.strip() else {}


def run_jobs(request, repository, run_id):
    jobs = []
    page = 1
    while True:
        result = request('GET', f'repos/{repository}/actions/runs/{run_id}/jobs?filter=all&per_page=100&page={page}')
        jobs.extend(result['jobs'])
        if len(result['jobs']) < 100:
            return jobs
        page += 1


def latest_jobs(jobs):
    return {job['name']: job for job in sorted(jobs, key=lambda job: (job['run_attempt'], job['id']))}


def required(job):
    return job['name'] != 'test' and job['name'] not in OPTIONAL_JOBS


def resume_plan(jobs, attempt):
    current = [job for job in jobs if job['run_attempt'] == attempt]
    if not current or any(job['status'] != 'completed' or job['conclusion'] == 'cancelled' for job in current):
        return []
    latest = [job for job in latest_jobs(jobs).values() if required(job)]
    if any(job['status'] != 'completed' or job['conclusion'] not in ('success', 'cancelled', 'skipped', 'neutral') for job in latest):
        return []
    previous = latest_jobs([job for job in jobs if job['run_attempt'] < attempt])
    repaired = [job for job in current if required(job) and job['conclusion'] == 'success'
                and previous.get(job['name'], {}).get('conclusion') in FAILED]
    return [job['name'] for job in latest if job['conclusion'] == 'cancelled'] if repaired else []


def same_attempt(run, event):
    return (run['event'] == 'pull_request' and run['status'] == 'completed'
            and run['run_attempt'] == event['run_attempt']
            and run['head_sha'] == event['head_sha'])


def resume(request, repository, payload):
    event = payload['workflow_run']
    if event['event'] != 'pull_request' or event['run_attempt'] < 2:
        return {'resumed': False, 'reason': 'not a PR retry'}
    run_id = int(event['id'])
    path = f'repos/{repository}/actions/runs/{run_id}'
    run = request('GET', path)
    if not same_attempt(run, event):
        return {'resumed': False, 'reason': 'attempt changed or still running'}
    remaining = resume_plan(run_jobs(request, repository, run_id), event['run_attempt'])
    if not remaining:
        return {'resumed': False, 'reason': 'no repaired failure with cancelled siblings'}
    pulls = run.get('pull_requests') or request('GET', f'repos/{repository}/commits/{run["head_sha"]}/pulls?per_page=100')
    current_pr = False
    for pull in pulls:
        pull = request('GET', f'repos/{repository}/pulls/{int(pull["number"])}')
        if pull['state'] == 'open' and pull['head']['sha'] == run['head_sha']:
            current_pr = True
            break
    if not current_pr:
        return {'resumed': False, 'reason': 'PR closed or commit superseded'}
    if not same_attempt(request('GET', path), event):
        return {'resumed': False, 'reason': 'attempt changed before resume'}
    request('POST', path + '/rerun-failed-jobs')
    return {'resumed': True, 'run_id': run_id, 'after_attempt': event['run_attempt'], 'jobs': remaining}


def check(request, repository, run_id):
    jobs = [job for job in latest_jobs(run_jobs(request, repository, run_id)).values() if required(job)]
    incomplete = [job['name'] for job in jobs if job['status'] != 'completed' or job['conclusion'] != 'success']
    return {'complete': bool(jobs) and not incomplete, 'incomplete': incomplete, 'checked': len(jobs)}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('command', choices=('resume', 'check'))
    parser.add_argument('--run-id', type=int)
    args = parser.parse_args()
    repository = os.environ['GITHUB_REPOSITORY']
    if args.command == 'resume':
        with open(os.environ['GITHUB_EVENT_PATH']) as source:
            result = resume(api, repository, json.load(source))
    else:
        if not args.run_id:
            parser.error('check requires --run-id')
        result = check(api, repository, args.run_id)
    print(json.dumps(result))
    if args.command == 'check' and not result['complete']:
        raise SystemExit(1)


if __name__ == '__main__':
    main()
