import sys
import os

from .utils import ydump, mkdir_p


def main(cities_dir):
    data = {}
    data['cityid'] = input('cityid: ')
    data['cityname'] = input('cityname: ')
    data['coordinates'] = [
        float(input('lng: ')),
        float(input('lat: '))
    ]
    tf_location_id = input('tf_location_id: ')
    if tf_location_id:
        data['tf_location_ids'] = [tf_location_id]
    data['zoom'] = 12

    yaml_part = ydump(data).decode('utf-8')
    mkdir_p(os.path.join(cities_dir, data['cityid']))
    filename = os.path.join(cities_dir, data['cityid'],
                            data['cityid'] + '.md')
    contents = ['', yaml_part, '\n(c) [Name](http://)']
    contents = '---\n'.join(contents)
    with open(filename, 'w') as f:
        f.write(contents)


if __name__ == '__main__':
    main(sys.argv[1])
