pragma solidity 0.5.0;
// pragma experimental ABIEncoderV2;

import "./StringSetLib.sol";
import "./UserRegistry.sol";

contract Protocol
{
    using StringSetLib for StringSetLib.StringSet;

    struct Repo {
        bool exists;

        StringSetLib.StringSet refs;
        mapping(bytes32 => bytes20) refsToCommits;

        StringSetLib.StringSet admins;
        StringSetLib.StringSet pushers;
        StringSetLib.StringSet pullers;
        bool isPublic;
    }

    mapping(bytes32 => Repo) repositories;

    event LogSetUsername(address indexed addr, string indexed username);
    event LogCreateRepo(address indexed user, string indexed repoID);
    event LogDeleteRepo(address indexed user, string indexed repoID);
    event LogSetPublic(address indexed user, string repoID, bool isPublic);
    event LogUpdateRef(address indexed user, string indexed repoID, string indexed refName, bytes20 commitHash);
    event LogDeleteRef(address indexed user, string indexed repoID, string indexed refName);

    address public owner;
    UserRegistry public userRegistry;

    constructor() public {
        owner = msg.sender;
    }

    function setUserRegistry(address _userRegistry) public {
        require(msg.sender == owner);
        userRegistry = UserRegistry(_userRegistry);
    }

    function setUsername(string memory username) public {
        userRegistry.setUsername(msg.sender, username);
        emit LogSetUsername(msg.sender, username);
    }

    function usernamesByAddress(address addr) public view returns (string memory) {
        return userRegistry.usernamesByAddress(addr);
    }

    function createRepo(string memory repoID) public {
        // Ensure that repoID is nonempty
        require(bytes(repoID).length > 0, "argument 'repoID' cannot be empty");

        // Ensure that the user has registered a username
        string memory username = userRegistry.usernamesByAddress(msg.sender);
        require(bytes(username).length > 0, "you have not claimed a username");

        Repo storage repo = repositories[hashString(repoID)];

        // Ensure that the repo doesn't exist yet
        require(repo.exists == false, "this repoID has already been claimed");

        repo.exists = true;
        repo.pushers.add(username);
        repo.admins.add(username);
        repo.pullers.add(username);

        emit LogCreateRepo(msg.sender, repoID);
    }

    function deleteRepo(string memory repoID) public {
        require(addressIsAdmin(msg.sender, repoID), "you are not an admin of this repo");

        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists, "this repo does not exist");

        delete repositories[hashString(repoID)];

        emit LogDeleteRepo(msg.sender, repoID);
    }

    function setPublic(string memory repoID, bool isPublic) public {
        require(addressIsAdmin(msg.sender, repoID), "you are not an admin of this repo");

        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists, "this repo does not exist");

        repo.isPublic = isPublic;

        emit LogSetPublic(msg.sender, repoID, isPublic);
    }

    function updateRef(string memory repoID, string memory refName, bytes20 oldCommitHash, bytes20 newCommitHash) public {
        require(userHasPushAccess(userRegistry.usernamesByAddress(msg.sender), repoID), "you don't have push access");

        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists, "repo does not exist");
        require(oldCommitHash == repo.refsToCommits[hashString(refName)], "the provided oldCommitHash is incorrect");

        repo.refs.add(refName);
        repo.refsToCommits[hashString(refName)] = newCommitHash;

        emit LogUpdateRef(msg.sender, repoID, refName, newCommitHash);
    }

    function deleteRef(string memory repoID, string memory refName) public {
        require(userHasPushAccess(userRegistry.usernamesByAddress(msg.sender), repoID), "you don't have push access");

        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists, "repo does not exist");

        repo.refs.remove(refName);
        delete repo.refsToCommits[hashString(refName)];

        emit LogDeleteRef(msg.sender, repoID, refName);
    }

    function repoExists(string memory repoID) public view returns (bool) {
        return repositories[hashString(repoID)].exists;
    }

    function userIsAdmin(string memory username, string memory repoID) public view returns (bool) {
        return repositories[hashString(repoID)].admins.contains(username);
    }

    function userHasPullAccess(string memory username, string memory repoID) public view returns (bool) {
        bytes32 repoIDHash = hashString(repoID);
        if (repositories[repoIDHash].isPublic == true) {
            return true;
        }
        return repositories[repoIDHash].pullers.contains(username);
    }

    function userHasPushAccess(string memory username, string memory repoID) public view returns (bool) {
        return repositories[hashString(repoID)].pushers.contains(username);
    }

    function addressIsAdmin(address addr, string memory repoID) public view returns (bool) {
        return userIsAdmin(userRegistry.usernamesByAddress(addr), repoID);
    }

    function addressHasPullAccess(address addr, string memory repoID) public view returns (bool) {
        return userHasPullAccess(userRegistry.usernamesByAddress(addr), repoID);
    }

    function addressHasPushAccess(address addr, string memory repoID) public view returns (bool) {
        return userHasPushAccess(userRegistry.usernamesByAddress(addr), repoID);
    }

    function isRepoPublic(string memory repoID) public view returns(bool) {
        return repositories[hashString(repoID)].isPublic;
    }

    function getUserPermissions(string memory repoID, string memory username) public view returns (bool puller, bool pusher, bool admin) {
        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists, "repo does not exist");

        return (
            repo.pullers.contains(username),
            repo.pushers.contains(username),
            repo.admins.contains(username)
        );
    }

    function setUserPermissions(string memory repoID, string memory username, bool puller, bool pusher, bool admin) public {
        require(addressIsAdmin(msg.sender, repoID), "you are not an admin");

        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists, "repo does not exist");

        if (puller) {
            repo.pullers.add(username);
        } else {
            repo.pullers.remove(username);
        }

        if (pusher) {
            repo.pushers.add(username);
        } else {
            repo.pushers.remove(username);
        }

        if (admin) {
            repo.admins.add(username);
        } else {
            repo.admins.remove(username);
        }
    }

    function numRefs(string memory repoID) public view returns (uint) {
        return repositories[hashString(repoID)].refs.size();
    }

    function getRef(string memory repoID, string memory refName) public view returns (bytes20) {
        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists, "repo does not exist");

        return repo.refsToCommits[hashString(refName)];
    }

    function getRefs(string memory repoID, uint pageSize, uint page) public view returns (uint total, bytes memory data) {
        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists, "repo does not exist");

        string memory refName;
        bytes20 commit;
        uint len = 0;
        uint start = page * pageSize;
        for (uint i = 0; i < pageSize && i + start < repo.refs.size(); i++) {
            refName = repo.refs.get(i + start);
            commit = repo.refsToCommits[hashString(refName)];
            len += 32 + bytes(refName).length + 32 + 20;
        }

        data = new bytes(len);
        uint written = 0;
        for (uint i = 0; i < pageSize && i + start < repo.refs.size(); i++) {
            refName = repo.refs.get(i + start);
            commit = repo.refsToCommits[hashString(refName)];

            writeUint(bytes(refName).length, data, written);
            written += 32;
            writeBytes(bytes(refName), data, written);
            written += bytes(refName).length;

            writeUint(20, data, written);
            written += 32;
            writeBytes20(commit, data, written);
            written += 20;
        }

        return (repo.refs.size(), data);
    }

    enum UserType {
        ADMIN, PULLER, PUSHER
    }

    function getRepoUsers(string memory repoID, UserType whichUsers, uint pageSize, uint page) public view returns (uint total, bytes memory data) {
        Repo storage repo = repositories[hashString(repoID)];
        require(repo.exists, "repo does not exist");

        StringSetLib.StringSet storage users = repo.admins;
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
        for (uint i = 0; i < pageSize && i + start < users.size(); i++) {
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

    function writeBytes(bytes memory src, bytes memory dest, uint offset) private pure {
        for (uint i = 0; i < src.length; i++) {
            dest[i + offset] = src[i];
        }
    }

    function writeBytes20(bytes20 src, bytes memory dest, uint offset) private pure {
        for (uint8 i = 0; i < 20; i++) {
            dest[i + offset] = src[i];
        }
    }

    function hashString(string memory s) private pure returns (bytes32) {
        return keccak256(abi.encodePacked(s));
    }
}