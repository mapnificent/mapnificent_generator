import errno
import re
import os

import yaml

NON_WORD_RE = re.compile('[^\w-]')


def slugify(s):
    return NON_WORD_RE.sub('-', s)


def mkdir_p(path):
    try:
        os.makedirs(path)
    except OSError as exc:
        if exc.errno == errno.EEXIST and os.path.isdir(path):
            pass
        else:
            raise


def ydump(e):
    return yaml.safe_dump(e, allow_unicode=True, default_flow_style=False, encoding='utf-8', width=10000)
