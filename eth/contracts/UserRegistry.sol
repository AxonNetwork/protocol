pragma solidity ^0.5.0;
// pragma experimental ABIEncoderV2;

contract UserRegistry
{
    mapping(address => string) public usernamesByAddress;
    mapping(bytes32 => address) public addressesByUsername;

    constructor() public {
    }

    function setUsername(address addr, string memory username) public {
        require(bytes(username).length > 0, "argument 'username' cannot be empty");
        require(bytes(usernamesByAddress[addr]).length == 0, "your address already belongs to a username");

        bytes32 usernameHash = hashString(username);
        require(addressesByUsername[usernameHash] == address(0x0), "this username is already claimed");

        usernamesByAddress[addr] = username;
        addressesByUsername[usernameHash] = addr;
    }

    function getAddressForUsername(string memory username) public view returns (address) {
        return addressesByUsername[hashString(username)];
    }

    function hashString(string memory s) private pure returns (bytes32) {
        return keccak256(abi.encodePacked(s));
    }
}