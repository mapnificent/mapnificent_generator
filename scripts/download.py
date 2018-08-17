from datetime import datetime, timedelta
import os
import sys
import subprocess
import hashlib
import re
from io import BytesIO
from contextlib import closing

import yaml
import requests
from tqdm import tqdm

from .utils import ydump, mkdir_p, slugify


TRANSITFEEDS_API_URL = 'https://api.transitfeeds.com/v1/'


def get_transitfeeds_locations(location_ids):
    for location_id in location_ids:
        response = requests.get(TRANSITFEEDS_API_URL + 'getFeeds', params={
            'key': os.environ.get('TRANSITFEED_API_KEY'),
            'location': location_id,
            'descendants': 1,
            'page': 1,
            'limit': 50,
            'type': 'gtfs'
        })
        for feed in response.json()['results']['feeds']:
            yield feed['id']


def download_transitfeeds_feed(feed_id):
    try:
        return download_url(TRANSITFEEDS_API_URL + 'getLatestFeedVersion', params={
            'key': os.environ.get('TRANSITFEED_API_KEY'),
            'feed': feed_id
        })
    except ValueError:
        return None, None


def download_with_script(script, basepath):
    print('Starting script', script)
    proc = subprocess.Popen(script, shell=True, cwd=basepath,
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    outs, errs = proc.communicate()
    out_file = outs.decode('utf-8').splitlines()[-1]
    print('Saving', out_file)
    out_file = os.path.join(basepath, out_file)
    with open(out_file, 'rb') as f:
        return BytesIO(f.read()), None


def download_url(url, params=None):
    response_content = BytesIO()
    with closing(requests.get(url, params=params, stream=True)) as response:
        if response.headers.get('Content-Type') == 'application/json':
            raise ValueError('API error')
        size = None
        if response.headers.get('Content-Length'):
            size = int(response.headers.get('Content-Length').strip())
        with tqdm(total=size, unit='B', unit_scale=True) as pbar:
            for buf in response.iter_content(1024):
                response_content.write(buf)
                pbar.update(len(buf))
        url = response.url
    return response_content, url


def main(path):
    basepath = os.path.abspath(path)
    cityid = os.path.basename(basepath)
    filename = os.path.join(basepath, '{cityid}.md'.format(cityid=cityid))
    if not os.path.exists(filename):
        print('Filename does not exist', filename)
        return
    with open(filename) as f:
        for doc in yaml.load_all(f):
            data = doc
            break

    data_path = os.path.join(path, 'data')
    mkdir_p(data_path)

    now = datetime.utcnow()
    age = timedelta(days=7)

    new_files = False

    tf_location_ids = data.get('tf_location_ids')
    if tf_location_ids is not None:
        data.setdefault('gtfs', {})
        feed_ids = get_transitfeeds_locations(tf_location_ids)
        for feed_id in feed_ids:
            slug_feed_id = slugify(feed_id)
            data['gtfs'][slug_feed_id] = {'tf_feed_id': feed_id}

    if 'gtfs' not in data:
        print('No GTFS key', filename)
        return

    gtfs = data['gtfs']
    if not gtfs:
        print('No GTFS feeds present', filename)

    for key in gtfs.keys():
        print('Checking', key)
        info = gtfs[key]
        if isinstance(info, str):
            gtfs[key] = {'url': info}
            info = gtfs[key]

        current_hash_sum = info.get('sha256')
        data_file_path = os.path.join(data_path, '{}.zip'.format(key))
        if os.path.exists(data_file_path):
            statinfo = os.stat(data_file_path)
            last_modified = datetime.fromtimestamp(statinfo.st_mtime)
            if last_modified + age > now:
                print(f'Skipping {key}, less than 7 days old...')
                continue

        if info.get('script') is not None:
            response_content, final_url = download_with_script(info['script'], basepath)
        elif info.get('tf_feed_id') is not None:
            response_content, final_url = download_transitfeeds_feed(info['tf_feed_id'])
        elif info.get('url'):
            url = info['url']
            response_content, final_url = download_url(url)
        elif info.get('file'):
            final_url = None
            with open(data_file_path, 'rb') as f:
                response_content = BytesIO(f.read())
        else:
            print(f'Cannot update {key}, skipping...')
            continue

        if response_content is None:
            print(f'Update {key} failed, skipping...')
            continue

        if final_url is not None and not info.get('url'):
            # Keep permalinks that redirect
            info['url'] = final_url

        hashsum = hashlib.sha256()
        hashsum.update(response_content.getvalue())
        hexsum = hashsum.hexdigest()
        if current_hash_sum != hexsum or not os.path.exists(data_file_path):
            print(hexsum)
            info['sha256'] = hexsum
            new_files = True
            print('Saving', key)
            with open(data_file_path, 'wb') as f:
                f.write(response_content.getvalue())

    if new_files or data.get('northwest'):
        # Cleanup old meta data fields
        data.pop('northwest', None)
        data.pop('southeast', None)
        lat = data.pop('lat', None)
        lng = data.pop('lng', None)
        if lat is not None:
            data['coordinates'] = [lng, lat]
        if not data.get('hidden', True):
            data.pop('hidden', None)
        if data.get('active', False):
            data.pop('active', None)

        if 'added' not in data:
            data['added'] = now.isoformat()
        data['changed'] = now.isoformat()
        data['version'] = data.get('version', 0) + 1

        yaml_part = ydump(data).decode('utf-8')
        with open(filename) as f:
            contents = f.read()
        parts = contents.split('---\n')
        parts[1] = yaml_part
        contents = '---\n'.join(parts)
        with open(filename, 'w') as f:
            f.write(contents)

    outpath = os.path.join(basepath, f'{cityid}.bin')

    if data.get('script') is not None:
        print('Applying post download script')
        subprocess.run(
            [data['script']],
            stdout=sys.stdout,
            stderr=sys.stderr,
            cwd=basepath
        )

    if new_files or not os.path.exists(outpath):
        print('Calling mapnificent generator')
        subprocess.run(["./mapnificent_generator", "-d", data_path,
            "-o", outpath, '-v'], stdout=sys.stdout, stderr=sys.stderr)


if __name__ == '__main__':
    main(sys.argv[1])
