

with open('/data/in') as _in:
    with open('/data/out', 'w') as _out:

        data = _in.read(1024)
        print('got data ~> %s' % str(data))
        _out.write(data.upper())




