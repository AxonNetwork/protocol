pragma solidity ^0.4.24;
// pragma experimental ABIEncoderV2;

import "./strings.sol";

contract Protocol
{
    using strings for *;

    mapping(address => string) public usernamesByAddress;
    mapping(bytes32 => address) public addressesByUsername;

    struct Repo {
        bool exists;

        mapping(bytes32 => string) refs;
        string[] refsList;

        mapping(bytes32 => bool) pushers;
        string[] pushersList;

        bool isPrivate;
        mapping(bytes32 => bool) pullers;
        string[] pullersList;
    }

    mapping(bytes32 => Repo) repositories;

    event LogSetUsername(address addr, string username);

    constructor()
        public
    {

    }

    function setUsername(string username)
        public
    {
        require(bytes(username).length > 0);
        require(bytes(usernamesByAddress[msg.sender]).length == 0);

        bytes32 usernameHash = hashString(username);
        require(addressesByUsername[usernameHash] == 0x0);

        usernamesByAddress[msg.sender] = username;
        addressesByUsername[usernameHash] = msg.sender;

        emit LogSetUsername(msg.sender, username);
    }

    function createRepository(string repoID)
        public
    {
        require(bytes(repoID).length > 0);

        string memory username = usernamesByAddress[msg.sender];
        require(bytes(username).length > 0);

        bytes32 repoIDHash = hashString(repoID);
        bytes32 usernameHash = hashString(username);

        require(repositories[repoIDHash].exists == false);

        repositories[repoIDHash].exists = true;
        repositories[repoIDHash].pushersList.push(username);
        repositories[repoIDHash].pushers[usernameHash] = true;
    }

    function userHasPullAccess(string username, string repoID)
        public
        view
        returns (bool)
    {
        bytes32 repoIDHash = hashString(repoID);
        if (repositories[repoIDHash].isPrivate == false) {
            return true;
        }

        bytes32 usernameHash = hashString(username);
        return repositories[repoIDHash].pullers[usernameHash];
    }

    function userHasPushAccess(string username, string repoID)
        public
        view
        returns (bool)
    {
        bytes32 repoIDHash = keccak256(abi.encodePacked(repoID));
        bytes32 usernameHash = keccak256(abi.encodePacked(username));
        return repositories[repoIDHash].pushers[usernameHash];
    }

    function updateRef(string repoID, string refName, string commitHash)
        public
    {
        require(userHasPushAccess(usernamesByAddress[msg.sender], repoID));

        bytes32 repoIDHash = hashString(repoID);
        Repo storage repo = repositories[repoIDHash];

        bytes32 refNameHash = hashString(refName);
        if (bytes(repo.refs[refNameHash]).length == 0) {
            repo.refsList.push(refName);
        }
        repo.refs[refNameHash] = commitHash;
    }

    function numRefs(string repoID)
        public
        view
        returns (uint)
    {
        bytes32 repoIDHash = hashString(repoID);
        return repositories[repoIDHash].refsList.length;
    }

    function getRefs(string repoID, uint page)
        public
        view
        returns (string)
    {
        bytes32 repoIDHash = hashString(repoID);
        Repo storage repo = repositories[repoIDHash];
        require(repo.exists);

        string[] memory refNames = new string[](10);
        string[] memory commits = new string[](10);
        uint start = page * 10;
        for (uint i = 0; i + start < repo.refsList.length; i++) {
            bytes32 refNameHash = hashString(repo.refsList[i + start]);
            refNames[i] = repo.refsList[i + start];
            commits[i] = repo.refs[refNameHash];
        }

        strings.slice[] memory joined = new strings.slice[](10);
        for (i = 0; i < 10; i++) {
            strings.slice[] memory parts = new strings.slice[](2);
            parts[0] = refNames[i].toSlice();
            parts[1] = commits[i].toSlice();
            joined[i] = ":".toSlice().join(parts).toSlice();
        }

        return "/".toSlice().join(joined);
    }

    function hashString(string s)
        private
        pure
        returns (bytes32)
    {
        return keccak256(abi.encodePacked(s));
    }
}