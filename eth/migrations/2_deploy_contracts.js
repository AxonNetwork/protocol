const Protocol = artifacts.require('./Protocol.sol')
const UserRegistry = artifacts.require('./UserRegistry.sol')
const StringSetLib = artifacts.require('./StringSetLib.sol')

module.exports = function(deployer, accounts, network) {
    let protocolContract,
        userRegistryContract

    deployer.then(() => {
        return deployer.deploy(StringSetLib)
    }).then(() => {
        return deployer.link(StringSetLib, Protocol)
    }).then(() => {
        return deployer.deploy(Protocol)
    }).then(() => {
        return deployer.deploy(UserRegistry)
    }).then(() => {
        return Protocol.deployed()
    }).then(p => {
        protocolContract = p
        return UserRegistry.deployed()
    }).then(ur => {
        userRegistryContract = ur
        return protocolContract.setUserRegistry(ur.address)
    })

    // 0. deploy contracts
    // 1. call Protocol.setUserRegistry()
    // 2. set usernames for jupiter/saturn
    //     - 0x54C9e13a2F5D6F850a1B31b9308d078B9c266602 (jupiter)
    //     - 0x22eeeb343cDa44A0c0e15Ba7a4a0C3D2657CDd56 (saturn)
}
