// const ConvertLib = artifacts.require('./ConvertLib.sol')
const Protocol = artifacts.require('./Protocol.sol')

module.exports = function(deployer, accounts, network) {
    // deployer.deploy(ConvertLib)
    // deployer.link(ConvertLib, Protocol)
    deployer.deploy(Protocol)
}
