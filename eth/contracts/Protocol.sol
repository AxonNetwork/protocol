pragma solidity ^0.4.24;
// pragma experimental ABIEncoderV2;

import "./StringSetLib.sol";

contract Protocol
{
    using StringSetLib for StringSetLib.StringSet;

    mapping(address => string) public usernamesByAddress;
    mapping(bytes32 => address) public addressesByUsername;

    struct Repo {
        bool exists;

        StringSetLib.StringSet refs;
        mapping(bytes32 => string) refsToCommits;

        StringSetLib.StringSet admins;
        StringSetLib.StringSet pushers;
        StringSetLib.StringSet pullers;
        bool isPrivate;
    }

    mapping(bytes32 => Repo) repositories;

    event LogSetUsername(address indexed addr, string indexed username);
    event LogCreateRepo(address indexed user, string indexed repoID);
    event LogDeleteRepo(address indexed user, string indexed repoID);
    event LogUpdateRef(address indexed user, string indexed repoID, string indexed refName, string commitHash);
    event LogDeleteRef(address indexed user, string indexed repoID, string indexed refName);
    event LogSetAdmin(address indexed user, string indexed username, string indexed repoID, bool isAdmin);

    constructor() public {
    }

    function setUsername(string username) public {
        require(bytes(username).length > 0);
        require(bytes(usernamesByAddress[msg.sender]).length == 0);

        bytes32 usernameHash = hashString(username);
        require(addressesByUsername[usernameHash] == 0x0);

        usernamesByAddress[msg.sender] = username;
        addressesByUsername[usernameHash] = msg.sender;

        emit LogSetUsername(msg.sender, username);
    }

    function getAddressForUsername(string username) public view returns (address) {
        return addressesByUsername[hashString(username)];
    }

    function createRepo(string repoID) public {
        // Ensure that repoID is nonempty
        require(bytes(repoID).length > 0);

        // Ensure that the user has registered a username
        string memory username = usernamesByAddress[msg.sender];
        require(bytes(username).length > 0);

        Repo storage repo = repositories[hashString(repoID)];

        // Ensure that the repo doesn't exist yet
        require(repo.exists == false);

        repo.exists = true;
        repo.pushers.add(username);
        repo.admins.add(username);
        repo.pullers.add(username);

        emit LogCreateRepo(msg.sender, repoID);
    }

    function deleteRepo(string repoID) public {
        require(addressIsAdmin(msg.sender, repoID));

        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists);

        delete repositories[hashString(repoID)];

        emit LogDeleteRepo(msg.sender, repoID);
    }

    function setAdmin(string username, string repoID, bool isAdmin) public returns (bool) {
        require(userIsAdmin(usernamesByAddress[msg.sender], repoID) != isAdmin);

        Repo storage repo = repositories[hashString(repoID)];
        if (isAdmin) {
            repo.admins.add(username);
        } else {
            repo.admins.remove(username);
        }
        emit LogSetAdmin(msg.sender, username, repoID, isAdmin);
    }

    function updateRef(string repoID, string refName, string commitHash) public {
        require(userHasPushAccess(usernamesByAddress[msg.sender], repoID));
        require(bytes(commitHash).length == 40);

        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists);

        repo.refs.add(refName);
        repo.refsToCommits[hashString(refName)] = commitHash;

        emit LogUpdateRef(msg.sender, repoID, refName, commitHash);
    }

    function deleteRef(string repoID, string refName) public {
        require(userHasPushAccess(usernamesByAddress[msg.sender], repoID));

        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists);

        repo.refs.remove(refName);
        delete repo.refsToCommits[hashString(refName)];

        emit LogDeleteRef(msg.sender, repoID, refName);
    }

    function repoExists(string repoID) public view returns (bool) {
        return repositories[hashString(repoID)].exists;
    }

    function userIsAdmin(string username, string repoID) public view returns (bool) {
        return repositories[hashString(repoID)].admins.contains(username);
    }

    function userHasPullAccess(string username, string repoID) public view returns (bool) {
        bytes32 repoIDHash = hashString(repoID);
        if (repositories[repoIDHash].isPrivate == false) {
            return true;
        }
        return repositories[repoIDHash].pullers.contains(username);
    }

    function userHasPushAccess(string username, string repoID) public view returns (bool) {
        return repositories[hashString(repoID)].pushers.contains(username);
    }

    function addressIsAdmin(address addr, string repoID) public view returns (bool) {
        return userIsAdmin(usernamesByAddress[addr], repoID);
    }

    function addressHasPullAccess(address addr, string repoID) public view returns (bool) {
        return userHasPullAccess(usernamesByAddress[addr], repoID);
    }

    function addressHasPushAccess(address addr, string repoID) public view returns (bool) {
        return userHasPushAccess(usernamesByAddress[addr], repoID);
    }

    function numRefs(string repoID) public view returns (uint) {
        return repositories[hashString(repoID)].refs.size();
    }

    function getRef(string repoID, string refName) public view returns (string) {
        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists);

        return repo.refsToCommits[hashString(refName)];
    }

    function getRefs(string repoID, uint pageSize, uint page) public view returns (uint total, bytes data) {
        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists);

        string memory refName;
        string memory commit;
        uint len = 0;
        uint start = page * pageSize;
        for (uint i = 0; i < pageSize && i + start < repo.refs.size(); i++) {
            refName = repo.refs.get(i + start);
            commit = repo.refsToCommits[hashString(refName)];
            len += 32 + bytes(refName).length + 32 + bytes(commit).length;
        }

        data = new bytes(len);
        uint written = 0;
        for (i = 0; i < pageSize && i + start < repo.refs.size(); i++) {
            refName = repo.refs.get(i + start);
            commit = repo.refsToCommits[hashString(refName)];

            writeUint(bytes(refName).length, data, written);
            written += 32;
            writeBytes(bytes(refName), data, written);
            written += bytes(refName).length;

            writeUint(bytes(commit).length, data, written);
            written += 32;
            writeBytes(bytes(commit), data, written);
            written += bytes(commit).length;
        }

        return (repo.refs.size(), data);
    }

    enum UserType {
        ADMIN, PULLER, PUSHER
    }

    function getRepoUsers(string repoID, UserType whichUsers, uint pageSize, uint page) public view returns (uint total, bytes data) {
        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists);

        StringSetLib.StringSet storage users;
        if (whichUsers == UserType.ADMIN) {
            users = repo.admins;
        } else if (whichUsers == UserType.PULLER) {
            users = repo.pullers;
        } else if (whichUsers == UserType.PUSHER) {
            users = repo.pushers;
        }

        string memory user;
        uint len = 0;
        uint start = page * pageSize;
        for (uint i = 0; i < pageSize && i + start < users.size(); i++) {
            user = users.get(i + start);
            len += 32 + bytes(user).length;
        }

        data = new bytes(len);
        uint written = 0;
        for (i = 0; i < pageSize && i + start < users.size(); i++) {
            user = users.get(i + start);

            writeUint(bytes(user).length, data, written);
            written += 32;
            writeBytes(bytes(user), data, written);
            written += bytes(user).length;
        }

        return (users.size(), data);
    }

    function writeUint(uint x, bytes memory dest, uint offset) private pure {
        bytes32 b = bytes32(x);
        for (uint i = 0; i < 32; i++) {
            dest[i + offset] = b[i];
        }
    }

    function writeBytes(bytes src, bytes dest, uint offset) private pure {
        for (uint i = 0; i < src.length; i++) {
            dest[i + offset] = src[i];
        }
    }

    function hashString(string s) private pure returns (bytes32) {
        return keccak256(abi.encodePacked(s));
    }
}