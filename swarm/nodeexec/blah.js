
const fs = require('fs')

const _in = fs.createReadStream('/data/in')
const _out = fs.createWriteStream('/data/out')

_in.on('data', data => {
    console.log('got some data ~>', data)
    _out.write('here was your input: <'+data.toString()+'>')
    _out.end()
})





