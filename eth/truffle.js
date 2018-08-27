var HDWalletProvider = require('truffle-hdwallet-provider')

// defaults
var conf = {
  mnemonic: "",
  host: ""
}
var appCfg = require('rc')('conscience', conf);
var mnemonic = appCfg.node.EthereumBIP39Seed
var host = appCfg.node.EthereumHost

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
      // from: '0xCE1f819Af3447C59676bc03B119721AbCD40EFBE',
    },
    localgeth: {
      network_id: 23332,
      provider: function() {
          return new HDWalletProvider(mnemonic, host)
      },
      gas: 4000000,
      gasPrice: 50000000000,
    },
    awsgeth: {
      network_id: 23332,
      provider: function() {
          return new HDWalletProvider(mnemonic, host)
      },
      gas: 4000000,
      gasPrice: 50000000000,
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
      port: 8545,
      network_id: 4,
      provider: function() {
          return new HDWalletProvider(mnemonic, host)
      },
      gas: 4000000,
    },
  },
  migrations_directory: './migrations'
}
