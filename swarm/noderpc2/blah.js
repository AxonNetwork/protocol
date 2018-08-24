const PROTO_PATH = __dirname + '/noderpc.proto'

const protoLoader = require('@grpc/proto-loader')
const grpcLibrary = require('grpc')

const packageDefinition = protoLoader.loadSync(PROTO_PATH, {})
const packageObject = grpcLibrary.loadPackageDefinition(packageDefinition)

const noderpc = packageObject.noderpc
const client = new noderpc.NodeRPC('localhost:1337', grpcLibrary.credentials.createInsecure())

client.hello({ name: 'brynskies' }, (err, asdf) => {
    console.log('err', err)
    console.log('asdf', asdf)
})

const asdf = client.helloStream({ name: 'brynskies' })

asdf.on('data', (resp) => {
    console.log('data ~>', resp)
})
