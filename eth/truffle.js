const HDWalletProvider = require('truffle-hdwallet-provider')

module.exports = {
  /**
   * Networks define how you connect to your ethereum client and let you set the
   * defaults web3 uses to send transactions. If you don't specify one truffle
   * will spin up a development blockchain for you on port 9545 when you
   * run `develop` or `test`. You can ask a truffle command to use a specific
   * network from the command line, e.g
   *
   * $ truffle test --network <network-name>
   */

  networks: {
    // Useful for testing. The `development` name is special - truffle uses it by default
    // if it's defined here and no other network is specified at the command line.
    // You should run a client (like ganache-cli, geth or parity) in a separate terminal
    // tab if you use this network and you must also set the `host`, `port` and `network_id`
    // options below to some value.
    //
    // development: {
    //  host: "127.0.0.1",     // Localhost (default: none)
    //  port: 8545,            // Standard Ethereum port (default: none)
    //  network_id: "*",       // Any network (default: none)
    // },

    // Another network with more advanced options...
    // advanced: {
      // port: 8777,             // Custom port
      // network_id: 1342,       // Custom network
      // gas: 8500000,           // Gas sent with each transaction (default: ~6700000)
      // gasPrice: 20000000000,  // 20 gwei (in wei) (default: 100 gwei)
      // from: <address>,        // Account to send txs from (default: accounts[0])
      // websockets: true        // Enable EventEmitter interface for web3 (default: false)
    // },

    // Useful for deploying to a public network.
    // NB: It's important to wrap the provider as a function.
    kaleido: {
        provider: () => new HDWalletProvider(process.env.DEPLOYMENT_MNEMONIC, `https://${process.env.KALEIDO_USERNAME}:${process.env.KALEIDO_PASSWORD}@${process.env.KALEIDO_ETH_HOST_HTTPS}`),
        network_id: '*',
        gas: 8000000,
        gasPrice: 0,
        type: 'quorum',
        // confirmations: 2,    // # of confs to wait between deployments. (default: 0)
        // timeoutBlocks: 200,  // # of blocks before a deployment times out  (minimum/default: 50)
        // skipDryRun: true     // Skip dry run before migrations? (default: false for public nets )
    },
    awsgeth: {
      network_id: 23332,
      provider: () => new HDWalletProvider(process.env.MASTER_MNEMONIC, 'http://hera.axon.science:8545'),
      gas: 6900000,
      gasPrice: 0, //50000000000,
      // type: 'quorum',
      // websockets: true,
    },
    bloxberg: {
      network_id: 8995,
      provider: () => new HDWalletProvider(process.env.MASTER_MNEMONIC, 'https://bloxberg.org/eth/rpc'),
      gas: 6900000,
      gasPrice: 0, //50000000000,
      // type: 'quorum',
      // websockets: true,
    },
    rinkeby: {
      port: 8545,
      network_id: 4,
      provider: () => new HDWalletProvider(mnemonic, host),
      gas: 4000000,
    },

    // Useful for private networks
    // private: {
      // provider: () => new HDWalletProvider(mnemonic, `https://network.io`),
      // network_id: 2111,   // This network is yours, in the cloud.
      // production: true    // Treats this network as if it was a public net. (default: false)
    // }
  },

  // Set default mocha options here, use special reporters etc.
  mocha: {
    // timeout: 100000
  },

  // Configure your compilers
  compilers: {
    solc: {
      version: "0.5.0",    // Fetch exact version from solc-bin (default: truffle's version)
      // docker: true,        // Use "0.5.1" you've installed locally with docker (default: false)
      // settings: {          // See the solidity docs for advice about optimization and evmVersion
      //  optimizer: {
      //    enabled: false,
      //    runs: 200
      //  },
      //  evmVersion: "byzantium"
      // }
    }
  }
}



