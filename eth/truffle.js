// var HDWalletProvider = require('truffle-hdwallet-provider')

module.exports = {
  rpc: {
    host: 'localhost',
    port: 8545,
    gas: 4000000,
    // gas: 4000000,
  },
  networks: {
    dev: {
      host: "localhost",
      port: 8545,
      network_id: "*",
      gas: 4000000,
      from: '0xCE1f819Af3447C59676bc03B119721AbCD40EFBE',
    },
    mainnet: {
      network_id: 1,
      provider: function() {
          return new HDWalletProvider('12 words go here', 'https://mainnet.infura.io/API KEY HERE')
      },
      gas: 4000000,
      gasPrice: 50000000000,
    },
    mainnetgeth: {
      host: "localhost",
      port: 8545,
      network_id: 1,
      gas: 4000000,
      gasPrice: 50000000000,
    },
    ropsten: {
      port: 8545,
      network_id: 3,
      provider: function() {
          return new HDWalletProvider('12 words go here', 'https://ropsten.infura.io/API KEY HERE')
      },
      gas: 4000000,
    },
    rinkeby: {
      host: "localhost",
      port: 8545,
      network_id: 4,
    },
  },
  migrations_directory: './migrations'
}
