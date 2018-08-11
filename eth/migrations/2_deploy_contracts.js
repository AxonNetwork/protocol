// const ConvertLib = artifacts.require('./ConvertLib.sol')
const Protocol = artifacts.require('./Protocol.sol')
const StringSetLib = artifacts.require('./StringSetLib.sol')

module.exports = function(deployer, accounts, network) {
    deployer.deploy(StringSetLib)
    deployer.link(StringSetLib, Protocol)
    deployer.deploy(Protocol)
}
